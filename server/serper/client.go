// Package serper is the Serper places adapter. The key stays in memory and out of logs and errors.
package serper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/domain"
)

const (
	defaultURL = "https://google.serper.dev/places"
	maxPlaces  = 3
	maxBody    = 64 << 10
	timeout    = 10 * time.Second
)

type Client struct {
	Key, URL string
	Client   *http.Client
}

// FindPlaces runs one Serper places query and keeps at most three places.
func (c *Client) FindPlaces(ctx context.Context, query string) ([]assistant.Place, error) {
	started := time.Now()
	places, status, err := c.find(ctx, query)
	if err != nil {
		slog.Warn("place lookup failed", "query_len", len([]rune(query)), "status", status, "error", err, "duration_ms", time.Since(started).Milliseconds(), "trace_id", domain.TraceID(ctx))
		return nil, err
	}
	slog.Info("place lookup", "query_len", len([]rune(query)), "places", len(places), "status", status, "duration_ms", time.Since(started).Milliseconds(), "trace_id", domain.TraceID(ctx))
	return places, nil
}

func (c *Client) find(ctx context.Context, query string) ([]assistant.Place, int, error) {
	if c.Key == "" {
		return nil, 0, errors.New("the Serper key is not configured")
	}
	url := c.URL
	if url == "" {
		url = defaultURL
	}
	body, _ := json.Marshal(map[string]string{"q": query, "gl": "nl", "hl": "en"})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, errors.New("the lookup request could not be built")
	}
	req.Header.Set("X-API-KEY", c.Key)
	req.Header.Set("Content-Type", "application/json")
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, errors.New("the lookup service did not answer")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("the lookup service answered with status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, resp.StatusCode, errors.New("the lookup reply could not be read")
	}
	if len(raw) > maxBody {
		return nil, resp.StatusCode, errors.New("the lookup reply was too large")
	}
	var decoded struct {
		Places []struct {
			Title    string  `json:"title"`
			Address  string  `json:"address"`
			Category string  `json:"category"`
			Website  string  `json:"website"`
			Rating   float64 `json:"rating"`
		} `json:"places"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, resp.StatusCode, errors.New("the lookup reply was unreadable")
	}
	places := []assistant.Place{}
	for _, p := range decoded.Places {
		if len(places) == maxPlaces {
			break
		}
		places = append(places, assistant.Place{Title: p.Title, Address: p.Address, Category: p.Category, Website: p.Website, Rating: p.Rating})
	}
	return places, resp.StatusCode, nil
}
