// Package serper is the Serper search adapter. The key stays in memory and out of logs and errors.
package serper

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/domain"
)

const defaultBase = "https://google.serper.dev"

// Client implements assistant.Finder. URL is the base URL; empty means Serper.
type Client struct {
	Key, URL string
	Client   *http.Client
}

// Lookup runs one search of the given kind and keeps at most three hits.
func (c *Client) Lookup(ctx context.Context, kind assistant.LookupKind, query string) ([]assistant.Hit, error) {
	target, ok := endpoints[kind]
	if !ok {
		return nil, fmt.Errorf("unknown lookup kind %q", kind)
	}
	started := time.Now()
	body, status, err := c.transport().post(ctx, target.path, map[string]any{"q": query, "gl": "nl", "hl": "en"})
	var hits []assistant.Hit
	if err == nil {
		hits, err = target.decode(body)
	}
	c.log(ctx, kind, query, target.credits, status, len(hits), started, err)
	return hits, err
}

func (c *Client) transport() transport {
	return transport{key: c.Key, base: c.base(), http: c.Client}
}

func (c *Client) base() string {
	if c.URL == "" {
		return defaultBase
	}
	return c.URL
}

// log writes one line per lookup. It records sizes and costs, never the query, a hit or the key.
func (c *Client) log(ctx context.Context, kind assistant.LookupKind, query string, credits, status, results int, started time.Time, err error) {
	fields := []any{"kind", string(kind), "query_len", len([]rune(query)), "status", status, "duration_ms", time.Since(started).Milliseconds(), "trace_id", domain.TraceID(ctx)}
	if err != nil {
		slog.Warn("serper lookup failed", append(fields, "error", err)...)
		return
	}
	slog.Info("serper lookup", append(fields, "results", results, "credits", credits)...)
}
