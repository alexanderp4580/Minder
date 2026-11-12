package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

func NewID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}

type traceKey struct{}

// WithTrace attaches an identifier that ties the log lines of one request or reply together.
func WithTrace(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}

// TraceID returns the identifier set by WithTrace, or an empty string.
func TraceID(ctx context.Context) string {
	id, _ := ctx.Value(traceKey{}).(string)
	return id
}
