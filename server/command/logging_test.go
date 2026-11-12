package command

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
)

// syncBuffer lets the test read log output while background goroutines still write it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func capture(t *testing.T) func() []map[string]any {
	t.Helper()
	var buf syncBuffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return func() []map[string]any {
		var out []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			var r map[string]any
			if json.Unmarshal([]byte(line), &r) == nil {
				out = append(out, r)
			}
		}
		return out
	}
}
func first(records []map[string]any, msg string) map[string]any {
	for _, r := range records {
		if r["msg"] == msg {
			return r
		}
	}
	return nil
}

func TestChangeLifecycleIsLoggedWithoutTitles(t *testing.T) {
	logs := capture(t)
	c, _ := setup(t)
	p, err := c.Preview(domain.Proposal{Action: "create_item", Title: "PRIVATEPLUMBER", DueAt: "2026-11-12T10:00:00+01:00", RemindAt: "2026-11-12T09:00:00+01:00", NagMinutes: 15})
	if err != nil {
		t.Fatal(err)
	}
	pv := first(logs(), "change previewed")
	if pv == nil || pv["action"] != "create_item" || pv["change_id"] != p.Change.ID || pv["entry_id"] != p.ID || pv["fields"] == nil {
		t.Fatalf("preview not logged: %v", pv)
	}
	r, err := c.Confirm(p.Change.ID)
	if err != nil {
		t.Fatal(err)
	}
	cf := first(logs(), "change confirmed")
	if cf == nil || cf["change_id"] != p.Change.ID || cf["item_id"] == "" || cf["version"] != float64(1) || cf["undo_id"] != r.Change.UndoID || cf["action"] != "create_item" {
		t.Fatalf("confirm not logged: %v", cf)
	}
	if _, err = c.Confirm(p.Change.ID); err != nil {
		t.Fatal(err)
	}
	if rep := first(logs(), "change confirm repeated"); rep == nil || rep["change_id"] != p.Change.ID {
		t.Fatalf("idempotent confirm not logged: %v", rep)
	}
	if _, err = c.Undo(r.Change.UndoID); err != nil {
		t.Fatal(err)
	}
	if u := first(logs(), "change undone"); u == nil || u["audit_id"] != r.Change.UndoID || u["item_id"] == "" {
		t.Fatalf("undo not logged: %v", u)
	}
	for _, l := range logs() {
		b, _ := json.Marshal(l)
		if strings.Contains(string(b), "PRIVATEPLUMBER") {
			t.Fatalf("title reached the log: %s", b)
		}
	}
}

func TestDiscardConflictAndInvalidProposalsAreLogged(t *testing.T) {
	logs := capture(t)
	c, _ := setup(t)
	p, _ := c.Preview(domain.Proposal{Action: "create_item", Title: "T"})
	if _, err := c.Discard(p.Change.ID); err != nil {
		t.Fatal(err)
	}
	if d := first(logs(), "change discarded"); d == nil || d["change_id"] != p.Change.ID {
		t.Fatalf("discard not logged: %v", d)
	}
	if _, err := c.Confirm(p.Change.ID); err == nil {
		t.Fatal("confirmed a discarded change")
	}
	if r := first(logs(), "change confirm rejected"); r == nil || r["level"] != "WARN" || r["error_code"] != "conflict" {
		t.Fatalf("conflict not logged: %v", r)
	}
	if _, err := c.Preview(domain.Proposal{Action: "nonsense"}); err == nil {
		t.Fatal("bad action accepted")
	}
	if r := first(logs(), "change preview rejected"); r == nil || r["action"] != "nonsense" || r["error_code"] != "invalid_field" {
		t.Fatalf("invalid proposal not logged: %v", r)
	}
}

func TestPreviewAllAndRefreshAreLogged(t *testing.T) {
	logs := capture(t)
	c, _ := setup(t)
	out, err := c.PreviewAll([]domain.Proposal{{Action: "create_item", Title: "A"}, {Action: "create_item", Title: "B"}})
	if err != nil {
		t.Fatal(err)
	}
	if r := first(logs(), "changes previewed"); r == nil || r["count"] != float64(2) {
		t.Fatalf("batch not logged: %v", r)
	}
	if _, err = c.Refresh(out[0].Change.ID); err != nil {
		t.Fatal(err)
	}
	if r := first(logs(), "change refreshed"); r == nil || r["old_change_id"] != out[0].Change.ID || r["new_change_id"] == nil {
		t.Fatalf("refresh not logged: %v", r)
	}
}
