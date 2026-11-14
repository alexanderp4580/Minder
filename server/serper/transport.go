package serper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	maxBody = 64 << 10
	timeout = 10 * time.Second
)

// transport sends one authenticated JSON POST to Serper. It knows nothing about endpoints
// or results, and its errors never carry the key or the provider's body.
type transport struct {
	key, base string
	http      *http.Client
}

// post sends the payload to base+path and returns the body and status. A non-200 status
// returns the status with an error.
func (t transport) post(ctx context.Context, path string, payload map[string]any) ([]byte, int, error) {
	if t.key == "" {
		return nil, 0, errors.New("the Serper key is not configured")
	}
	encoded, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.base+path, bytes.NewReader(encoded))
	if err != nil {
		return nil, 0, errors.New("the lookup request could not be built")
	}
	req.Header.Set("X-API-KEY", t.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client().Do(req)
	if err != nil {
		return nil, 0, errors.New("the lookup service did not answer")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("the lookup service answered with status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, resp.StatusCode, errors.New("the lookup reply could not be read")
	}
	if len(body) > maxBody {
		return nil, resp.StatusCode, errors.New("the lookup reply was too large")
	}
	return body, resp.StatusCode, nil
}

func (t transport) client() *http.Client {
	if t.http != nil {
		return t.http
	}
	return &http.Client{Timeout: timeout}
}
