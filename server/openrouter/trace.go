package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

type Trace struct {
	dir   string
	until time.Time
}

func NewTrace(dir string, until time.Time) (*Trace, error) {
	if dir == "" || until.IsZero() {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, errors.New("agent trace directory could not be created")
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("agent trace directory must be a real directory")
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return nil, errors.New("agent trace directory could not be secured")
	}
	return &Trace{dir: dir, until: until}, nil
}

// RecordDiagnostic is the agent's optional recorder. Capture failures never
// change a turn and never copy private content into ordinary logs.
func (m *Client) RecordDiagnostic(ctx context.Context, event string, data any) {
	m.Trace.record(ctx, "", event, data)
}

func (t *Trace) record(ctx context.Context, callID, event string, data any) {
	if t == nil || !time.Now().Before(t.until) {
		return
	}
	if err := t.write(ctx, callID, event, data); err != nil {
		slog.Warn("agent trace write failed", "event", event, "call_id", callID, "trace_id", domain.TraceID(ctx))
	}
}

func (t *Trace) write(ctx context.Context, callID, event string, data any) error {
	now := time.Now().UTC()
	record := struct {
		Time    time.Time `json:"time"`
		Event   string    `json:"msg"`
		TraceID string    `json:"trace_id"`
		CallID  string    `json:"call_id,omitempty"`
		Data    any       `json:"data"`
	}{now, event, domain.TraceID(ctx), callID, data}
	f, err := os.CreateTemp(t.dir, ".capture-*") // 0600; no user data in the name.
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	encodeErr := json.NewEncoder(f).Encode(record)
	closeErr := f.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), filepath.Join(t.dir, now.Format("20060102T150405.000000000Z")+"-"+domain.NewID()+".json"))
}
