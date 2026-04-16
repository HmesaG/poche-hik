package hikvision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
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
		FaceLibType: "staticLib",
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

// DeleteFace deletes a person's face from the device
func (c *Client) DeleteFace(ctx context.Context, employeeNo string) error {
	type FaceDataDelCond struct {
		FaceLibType string `json:"faceLibType"`
		FDID        string `json:"FDID"`
		FPID        []string `json:"FPID"`
	}

	cond := FaceDataDelCond{
		FaceLibType: "staticLib",
		FDID:        "1",
		FPID:        []string{employeeNo},
	}

	body, err := json.Marshal(cond)
	if err != nil {
		return fmt.Errorf("marshal delete face cond: %w", err)
	}

	_, err = c.Do(ctx, "PUT", "/ISAPI/Intelligent/FDLib/FaceDataDelete?format=json", nil, body)
	return err
}
