package hikvision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// FaceDataRecord defines the metadata for registering a face
type FaceDataRecord struct {
	FaceLibType string `json:"faceLibType"` // "blackList", "staticLib", etc.
	FDID        string `json:"FDID"`        // Face Database ID
	FPID        string `json:"FPID"`        // Face Person ID (usually employeeNo)
}

// RegisterFace registers a face image for a specific person
func (c *Client) RegisterFace(ctx context.Context, employeeNo string, faceImage []byte) error {
	// Usually for these terminals, FDID is "1" and FPID is the employeeNo
	record := FaceDataRecord{
		FaceLibType: "blackFD",
		FDID:        "1",
		FPID:        employeeNo,
	}

	recordData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal face record: %w", err)
	}

	// Create multipart body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Part 1: FaceDataRecord (JSON)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="FaceDataRecord";`)
	h.Set("Content-Type", "application/json")
	part, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("create face record part: %w", err)
	}
	part.Write(recordData)

	// Part 2: FaceImage (Binary)
	h = make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="FaceImage"; filename="face.jpg"`)
	h.Set("Content-Type", "image/jpeg")
	imagePart, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("create face image part: %w", err)
	}
	imagePart.Write(faceImage)

	writer.Close()
	path := "/ISAPI/Intelligent/FDLib/FaceDataRecord?format=json"

	// Perform the request
	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	_, err = c.Do(ctx, "POST", path, headers, body.Bytes())
	return err
}

// RegisterAccessControlFace attaches a face image to an access-control user.
// This is the endpoint reflected by "Face Pic. Number" in the Hikvision user UI.
func (c *Client) RegisterAccessControlFace(ctx context.Context, employeeNo string, faceImage []byte) error {
	type faceInfoRecord struct {
		EmployeeNo  string `json:"employeeNo"`
		FaceLibType string `json:"faceLibType,omitempty"`
		FDID        string `json:"FDID,omitempty"`
		FPID        string `json:"FPID,omitempty"`
	}

	var lastErr error
	records := []faceInfoRecord{
		{EmployeeNo: employeeNo},
		{EmployeeNo: employeeNo, FaceLibType: "blackFD", FDID: "1", FPID: employeeNo},
		{EmployeeNo: employeeNo, FaceLibType: "staticLib", FDID: "1", FPID: employeeNo},
	}
	methods := []string{"POST", "PUT"}

	for _, record := range records {
		for _, method := range methods {
			body, contentType, err := buildFaceMultipart("FaceInfo", record, faceImage)
			if err != nil {
				return err
			}
			_, err = c.Do(ctx, method, "/ISAPI/AccessControl/FaceInfo/Record?format=json",
				map[string]string{"Content-Type": contentType}, body)
			if err == nil {
				return nil
			}
			lastErr = err
		}
	}

	return fmt.Errorf("register access-control face %q: %w", employeeNo, lastErr)
}

func buildFaceMultipart(metaPartName string, metadata interface{}, faceImage []byte) ([]byte, string, error) {
	recordData, err := json.Marshal(metadata)
	if err != nil {
		return nil, "", fmt.Errorf("marshal face metadata: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s";`, metaPartName))
	h.Set("Content-Type", "application/json")
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, "", fmt.Errorf("create face metadata part: %w", err)
	}
	if _, err := part.Write(recordData); err != nil {
		return nil, "", err
	}

	h = make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="FaceImage"; filename="face.jpg"`)
	h.Set("Content-Type", "image/jpeg")
	imagePart, err := writer.CreatePart(h)
	if err != nil {
		return nil, "", fmt.Errorf("create face image part: %w", err)
	}
	if _, err := imagePart.Write(faceImage); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body.Bytes(), writer.FormDataContentType(), nil
}

// DownloadFace retrieves the face image from the device for a specific employee
func (c *Client) DownloadFace(ctx context.Context, employeeNo string) ([]byte, error) {
	// We use the same endpoint as Register but with GET
	path := fmt.Sprintf("/ISAPI/Intelligent/FDLib/FaceDataRecord?format=json&FDID=1&FPID=%s", employeeNo)

	resp, err := c.Do(ctx, "GET", path, nil, nil)
	if err == nil {
		if image, err := extractJPEG(resp); err == nil {
			return image, nil
		}
		if faceURL := findFaceImageURL(resp); faceURL != "" {
			return c.DownloadFaceURL(ctx, faceURL)
		}
	}

	if image, searchErr := c.downloadFaceFromFaceInfo(ctx, employeeNo); searchErr == nil {
		return image, nil
	} else if err != nil {
		if isFaceFeatureUnsupported(err) && isFaceFeatureUnsupported(searchErr) {
			return nil, fmt.Errorf("device does not support face download APIs")
		}
		return nil, fmt.Errorf("download face record: %w; face info search: %w", err, searchErr)
	}

	if len(resp) < 500 && strings.Contains(string(resp), "statusString") {
		return nil, fmt.Errorf("face not found or device error: %s", string(resp))
	}
	return nil, fmt.Errorf("no JPEG data or Hikvision local photo URL found in device response")
}

// DownloadFaceURL retrieves a Hikvision local photo URL such as:
// http://10.0.0.100/LOCALS/pic/enrlFace/0/0000000001.jpg@WEB000000000002
func (c *Client) DownloadFaceURL(ctx context.Context, rawURL string) ([]byte, error) {
	path, err := c.localPhotoPath(rawURL)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	return extractJPEG(resp)
}

func (c *Client) downloadFaceFromFaceInfo(ctx context.Context, employeeNo string) ([]byte, error) {
	payload := map[string]interface{}{
		"FaceInfoSearchCond": map[string]interface{}{
			"searchID":             fmt.Sprintf("face_%s", employeeNo),
			"searchResultPosition": 0,
			"maxResults":           1,
			"EmployeeNoList": []map[string]string{
				{"employeeNo": employeeNo},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal face info search: %w", err)
	}

	resp, err := c.Do(ctx, "POST", "/ISAPI/AccessControl/FaceInfo/Search?format=json",
		map[string]string{"Content-Type": "application/json"}, body)
	if err != nil {
		return nil, err
	}

	if image, err := extractJPEG(resp); err == nil {
		return image, nil
	}

	faceURL := findFaceImageURL(resp)
	if faceURL == "" {
		return nil, fmt.Errorf("face info search did not return a local photo URL")
	}
	return c.DownloadFaceURL(ctx, faceURL)
}

func (c *Client) localPhotoPath(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(strings.Trim(rawURL, `"`))
	if rawURL == "" {
		return "", fmt.Errorf("empty face URL")
	}

	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", fmt.Errorf("parse face URL: %w", err)
		}
		if parsed.Hostname() != "" && parsed.Hostname() != c.Host {
			log.Debug().Str("urlHost", parsed.Hostname()).Str("clientHost", c.Host).Msg("Downloading Hikvision face URL from configured device host")
		}
		if parsed.RawQuery != "" {
			return parsed.EscapedPath() + "?" + parsed.RawQuery, nil
		}
		return parsed.EscapedPath(), nil
	}

	if strings.HasPrefix(rawURL, "/") {
		return rawURL, nil
	}
	return "/" + rawURL, nil
}

func extractJPEG(resp []byte) ([]byte, error) {
	idx := bytes.Index(resp, []byte{0xFF, 0xD8, 0xFF})
	if idx == -1 {
		return nil, fmt.Errorf("no JPEG data found")
	}

	endIdx := bytes.LastIndex(resp, []byte{0xFF, 0xD9})
	if endIdx == -1 {
		return resp[idx:], nil
	}

	return resp[idx : endIdx+2], nil
}

func findFaceImageURL(resp []byte) string {
	body := string(resp)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`https?://[^"'<>\s]+/LOCALS/pic/enrlFace/[^"'<>\s]+\.jpe?g(?:@[^"'<>\s]+)?`),
		regexp.MustCompile(`/?LOCALS/pic/enrlFace/[^"'<>\s]+\.jpe?g(?:@[^"'<>\s]+)?`),
	}
	for _, pattern := range patterns {
		if match := pattern.FindString(body); match != "" {
			return strings.ReplaceAll(match, `\/`, `/`)
		}
	}
	return ""
}

// DeleteFace deletes a person's face from the device
func (c *Client) DeleteFace(ctx context.Context, employeeNo string) error {
	type FaceDataDelCond struct {
		FaceLibType string   `json:"faceLibType"`
		FDID        string   `json:"FDID"`
		FPID        []string `json:"FPID"`
	}

	cond := FaceDataDelCond{
		FaceLibType: "blackFD",
		FDID:        "1",
		FPID:        []string{employeeNo},
	}

	body, err := json.Marshal(cond)
	if err != nil {
		return fmt.Errorf("marshal delete face cond: %w", err)
	}

	_, err = c.Do(ctx, "PUT", "/ISAPI/Intelligent/FDLib/FaceDataDelete?format=json",
		map[string]string{"Content-Type": "application/json", "Accept": "application/json"}, body)
	return err
}

// UploadPhotoToHikvision updates the user photo on a Hikvision terminal using employeeNo as the key.
func (c *Client) UploadPhotoToHikvision(ctx context.Context, employeeNo string, photoData []byte) error {
	if err := c.RegisterFace(ctx, employeeNo, photoData); err == nil {
		return c.confirmFaceRegistered(ctx, employeeNo)
	} else if isFaceAlreadyExists(err) {
		log.Warn().Str("employeeNo", employeeNo).Str("device", c.Host).Msg("Foto ya existe en Hikvision, se intentara reemplazar")
	} else {
		log.Warn().Err(err).Str("employeeNo", employeeNo).Str("device", c.Host).Msg("Hikvision face upload failed, trying delete and recreate")
	}

	if delErr := c.DeleteFace(ctx, employeeNo); delErr == nil {
		if retryErr := c.RegisterFace(ctx, employeeNo, photoData); retryErr == nil {
			return c.confirmFaceRegistered(ctx, employeeNo)
		}
	} else {
		log.Warn().Err(delErr).Str("employeeNo", employeeNo).Str("device", c.Host).Msg("No se pudo eliminar la foto/cara anterior para reemplazarla")
	}

	if err := c.RegisterAccessControlFace(ctx, employeeNo, photoData); err == nil {
		return c.confirmFaceRegistered(ctx, employeeNo)
	} else {
		log.Warn().Err(err).Str("employeeNo", employeeNo).Str("device", c.Host).Msg("Hikvision FaceInfo upload failed, trying UserInfo fallback")
	}

	payload := map[string]interface{}{
		"UserInfo": map[string]string{
			"employeeNo": employeeNo,
			"photo":      base64.StdEncoding.EncodeToString(photoData),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal photo payload: %w", err)
	}

	headers := map[string]string{"Content-Type": "application/json"}
	if _, err := c.Do(ctx, "PUT", "/ISAPI/AccessControl/UserInfo/Update?format=json", headers, body); err != nil {
		log.Warn().Err(err).Str("employeeNo", employeeNo).Str("device", c.Host).Msg("Hikvision UserInfo photo update failed, trying face library")
		if faceErr := c.RegisterFace(ctx, employeeNo, photoData); faceErr != nil {
			if delErr := c.DeleteFace(ctx, employeeNo); delErr == nil {
				if retryErr := c.RegisterFace(ctx, employeeNo, photoData); retryErr == nil {
					return c.confirmFaceRegistered(ctx, employeeNo)
				} else if isFaceAlreadyExists(retryErr) {
					return c.confirmFaceRegistered(ctx, employeeNo)
				}
			}
			if isFaceAlreadyExists(faceErr) {
				return c.confirmFaceRegistered(ctx, employeeNo)
			}
			log.Warn().Err(faceErr).Str("employeeNo", employeeNo).Str("device", c.Host).Msg("Hikvision offline or photo update failed")
			return explainFaceUploadError(faceErr)
		}
		return c.confirmFaceRegistered(ctx, employeeNo)
	}

	return c.confirmFaceRegistered(ctx, employeeNo)
}

func isFaceAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "deviceUserAlreadyExistFace") || strings.Contains(msg, "alreadyExist")
}

func isFaceFeatureUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "notsupport") ||
		strings.Contains(msg, "notsupport") ||
		strings.Contains(msg, "not support") ||
		strings.Contains(msg, "methodnotallowed") ||
		strings.Contains(msg, "method not allowed")
}

func (c *Client) confirmFaceRegistered(ctx context.Context, employeeNo string) error {
	for attempt := 0; attempt < 3; attempt++ {
		user, err := c.GetUser(ctx, employeeNo)
		if err == nil && user != nil && user.NumOfFace > 0 {
			log.Info().Str("employeeNo", employeeNo).Str("device", c.Host).Int("numOfFace", user.NumOfFace).Msg("Foto sincronizada correctamente con Hikvision")
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
	return fmt.Errorf("device did not confirm face registration for employee %s", employeeNo)
}

func explainFaceUploadError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "picfeaturepoints"), strings.Contains(msg, "analysisfailed"):
		return fmt.Errorf("the device rejected the image because it could not model a valid face; use a frontal face photo with good lighting and a single visible face")
	case strings.Contains(msg, "multiplefaces"):
		return fmt.Errorf("the device rejected the image because it detected multiple faces")
	case strings.Contains(msg, "noface"):
		return fmt.Errorf("the device rejected the image because it could not detect a face")
	case strings.Contains(msg, "errorspictureresolution"), strings.Contains(msg, "facesizesmall"), strings.Contains(msg, "facesizebig"):
		return fmt.Errorf("the device rejected the image because of face size or resolution; try a closer portrait photo")
	default:
		return err
	}
}
