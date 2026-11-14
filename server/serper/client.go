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
	outcome{kind: kind, queryLen: len([]rune(query)), credits: target.credits, status: status, results: len(hits), elapsed: time.Since(started), err: err}.log(ctx)
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

// outcome is what one lookup leaves in the log: sizes and costs, never the query, a hit or the key.
type outcome struct {
	kind                               assistant.LookupKind
	queryLen, credits, status, results int
	elapsed                            time.Duration
	err                                error
}

func (o outcome) log(ctx context.Context) {
	fields := []any{"kind", string(o.kind), "query_len", o.queryLen, "status", o.status, "duration_ms", o.elapsed.Milliseconds(), "trace_id", domain.TraceID(ctx)}
	if o.err != nil {
		slog.Warn("serper lookup failed", append(fields, "error", o.err)...)
		return
	}
	slog.Info("serper lookup", append(fields, "results", o.results, "credits", o.credits)...)
}
