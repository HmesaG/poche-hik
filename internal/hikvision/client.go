package hikvision

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client handles communication with a Hikvision device using ISAPI and Digest Auth
type Client struct {
	Host     string
	Port     int
	Username string
	Password string
	HTTP     *http.Client
}

// NewClient creates a new ISAPI client
func NewClient(host string, port int, username, password string) *Client {
	return &Client{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		HTTP: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
}

// Do performs an HTTP request with Digest Authentication support
func (c *Client) Do(ctx context.Context, method, path string, headers map[string]string, body []byte) ([]byte, error) {
	url := c.BaseURL() + path
	log.Debug().Str("method", method).Str("url", url).Msg("Sending ISAPI request")

	// Helper to create request with headers
	newReq := func(b []byte) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		return req, nil
	}

	req, err := newReq(body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Handle Digest Auth
		authHeader := resp.Header.Get("WWW-Authenticate")
		if authHeader == "" {
			return nil, fmt.Errorf("unauthorized but no challenge header")
		}

		digestParams := parseDigestHeader(authHeader)
		digestAuth := c.calculateDigest(method, path, digestParams)

		// Re-send request with Authorization header
		req, err = newReq(body)
		if err != nil {
			return nil, fmt.Errorf("create authenticated request: %w", err)
		}
		req.Header.Set("Authorization", digestAuth)
		
		resp, err = c.HTTP.Do(req)
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
		return respBody, fmt.Errorf("error status: %d (%s)", resp.StatusCode, resp.Status)
	}

	return respBody, nil
}

// Internal helpers for Digest Auth
func parseDigestHeader(header string) map[string]string {
	params := make(map[string]string)
	if !strings.HasPrefix(header, "Digest ") {
		return params
	}
	
	header = header[7:]
	parts := strings.Split(header, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			key := kv[0]
			val := strings.Trim(kv[1], "\"")
			params[key] = val
		}
	}
	return params
}

func (c *Client) calculateDigest(method, path string, params map[string]string) string {
	realm := params["realm"]
	nonce := params["nonce"]
	qop := params["qop"]
	
	// HA1 = MD5(username:realm:password)
	h1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", c.Username, realm, c.Password)))
	ha1 := hex.EncodeToString(h1[:])
	
	// HA2 = MD5(method:digestURI)
	h2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, path)))
	ha2 := hex.EncodeToString(h2[:])
	
	var response string
	if qop == "auth" {
		nc := "00000001"
		cnonce := "abcdef0123456789" // In a real app, generate this randomly
		// response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
		h := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)))
		response = hex.EncodeToString(h[:])
		
		return fmt.Sprintf("Digest username=\"%s\", realm=\"%s\", nonce=\"%s\", uri=\"%s\", qop=%s, nc=%s, cnonce=\"%s\", response=\"%s\"",
			c.Username, realm, nonce, path, qop, nc, cnonce, response)
	}
	
	// Fallback for older digest
	h := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2)))
	response = hex.EncodeToString(h[:])
	return fmt.Sprintf("Digest username=\"%s\", realm=\"%s\", nonce=\"%s\", uri=\"%s\", response=\"%s\"",
		c.Username, realm, nonce, path, response)
}
