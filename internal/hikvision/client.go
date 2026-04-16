package hikvision

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	xmlContentType  = "application/xml; charset=UTF-8"
	defaultTimeout  = 20 * time.Second
	maxRetries      = 3
	retryBaseDelay  = 500 * time.Millisecond
)

// ISAPIError represents a structured error returned by the Hikvision ISAPI.
// It wraps the HTTP status code and the raw response body for caller inspection.
type ISAPIError struct {
	StatusCode int
	Body       []byte
}

func (e *ISAPIError) Error() string {
	return fmt.Sprintf("ISAPI error %d: %s", e.StatusCode, string(e.Body))
}

// Client handles communication with a Hikvision device using ISAPI and Digest Auth.
type Client struct {
	Host     string
	Port     int
	Username string
	Password string
	http     *http.Client
}

// NewClient creates a new ISAPI client for the given device.
func NewClient(host string, port int, username, password string) *Client {
	return &Client{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// BaseURL returns the base HTTP URL for the device.
func (c *Client) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
}

// Do performs an HTTP request with Digest Authentication and automatic retries.
// Retryable conditions: network errors and 5xx responses.
// Non-retryable: 4xx (except 401 which triggers Digest handshake).
func (c *Client) Do(ctx context.Context, method, path string, headers map[string]string, body []byte) ([]byte, error) {
	var (
		respBody []byte
		lastErr  error
	)

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * (1 << (attempt - 1)) // exponential: 500ms, 1s
			log.Debug().
				Int("attempt", attempt+1).
				Dur("delay", delay).
				Str("path", path).
				Msg("Retrying ISAPI request")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		respBody, lastErr = c.doOnce(ctx, method, path, headers, body)
		if lastErr == nil {
			return respBody, nil
		}

		// Only retry on network errors or 5xx; abort immediately on 4xx.
		if isAPIErr, ok := lastErr.(*ISAPIError); ok {
			if isAPIErr.StatusCode >= 400 && isAPIErr.StatusCode < 500 {
				return respBody, lastErr
			}
		}
	}

	return respBody, fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}

// doOnce executes a single attempt with Digest Auth challenge-response.
func (c *Client) doOnce(ctx context.Context, method, path string, headers map[string]string, body []byte) ([]byte, error) {
	fullURL := c.BaseURL() + path

	// Factory so we can replay the request after the Digest challenge.
	newReq := func() (*http.Request, error) {
		var bodyReader io.Reader
		if len(body) > 0 {
			bodyReader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return nil, err
		}
		// Default to XML unless the caller overrides Content-Type.
		if _, ok := headers["Content-Type"]; !ok && len(body) > 0 {
			req.Header.Set("Content-Type", xmlContentType)
		}
		req.Header.Set("Accept", "application/xml, text/xml, */*")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		return req, nil
	}

	// --- First probe (unauthenticated) to obtain the Digest challenge ---
	req, err := newReq()
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	log.Debug().Str("method", method).Str("url", fullURL).Msg("ISAPI request")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		authHeader := resp.Header.Get("WWW-Authenticate")
		if authHeader == "" {
			return nil, &ISAPIError{StatusCode: resp.StatusCode, Body: []byte("no WWW-Authenticate header")}
		}

		digestParams := parseDigestChallenge(authHeader)
		authorization, err := c.buildDigestAuth(method, path, digestParams)
		if err != nil {
			return nil, fmt.Errorf("build digest auth: %w", err)
		}

		// --- Second request with Authorization header ---
		req, err = newReq()
		if err != nil {
			return nil, fmt.Errorf("build authenticated request: %w", err)
		}
		req.Header.Set("Authorization", authorization)

		resp, err = c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("execute authenticated request: %w", err)
		}
		defer resp.Body.Close()
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn().
			Int("status", resp.StatusCode).
			Str("path", path).
			Str("body", string(respBody)).
			Msg("ISAPI non-2xx response")
		return respBody, &ISAPIError{StatusCode: resp.StatusCode, Body: respBody}
	}

	return respBody, nil
}

// parseDigestChallenge parses the WWW-Authenticate: Digest ... header into a key-value map.
func parseDigestChallenge(header string) map[string]string {
	params := make(map[string]string)
	if !strings.HasPrefix(header, "Digest ") {
		return params
	}
	// Split on commas, but be careful: values can contain commas inside quotes (rare in Hikvision, but safe).
	for _, part := range strings.Split(header[7:], ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			params[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
		}
	}
	return params
}

// buildDigestAuth computes the Authorization header value using RFC 2617 Digest Auth.
func (c *Client) buildDigestAuth(method, uri string, params map[string]string) (string, error) {
	realm := params["realm"]
	nonce := params["nonce"]
	qop := params["qop"]
	opaque := params["opaque"]
	algorithm := params["algorithm"] // usually "MD5" or empty

	_ = algorithm // We always use MD5 as Hikvision devices require it.

	// HA1 = MD5(username:realm:password)
	ha1 := hexMD5(fmt.Sprintf("%s:%s:%s", c.Username, realm, c.Password))

	// HA2 = MD5(method:uri)
	ha2 := hexMD5(fmt.Sprintf("%s:%s", method, uri))

	cnonce, err := randomHex(8)
	if err != nil {
		return "", fmt.Errorf("generate cnonce: %w", err)
	}

	const nc = "00000001"

	var response, authLine string
	if qop == "auth" || qop == "auth-int" {
		// response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
		response = hexMD5(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))
		authLine = fmt.Sprintf(
			`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%s"`,
			c.Username, realm, nonce, uri, qop, nc, cnonce, response,
		)
	} else {
		// Legacy: response = MD5(HA1:nonce:HA2)
		response = hexMD5(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
		authLine = fmt.Sprintf(
			`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
			c.Username, realm, nonce, uri, response,
		)
	}

	if opaque != "" {
		authLine += fmt.Sprintf(`, opaque="%s"`, opaque)
	}
	return authLine, nil
}

// hexMD5 returns the lowercase hex MD5 of the given string.
func hexMD5(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// randomHex generates a cryptographically secure random hex string of n bytes.
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
