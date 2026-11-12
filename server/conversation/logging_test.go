package conversation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/conversation"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
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
func wait(t *testing.T, logs func() []map[string]any, msg string) map[string]any {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if r := first(logs(), msg); r != nil {
			return r
		}
	}
	t.Fatalf("log %q never appeared", msg)
	return nil
}

type scripted struct {
	err   error
	panic any
	trace *string
}

func (s scripted) Reply(ctx context.Context, _ string) ([]domain.Entry, error) {
	if s.trace != nil {
		*s.trace = domain.TraceID(ctx)
	}
	if s.panic != nil {
		panic(s.panic)
	}
	return []domain.Entry{{Kind: "minder"}, {Kind: "preview"}}, s.err
}
func open(t *testing.T, r conversation.Replier) *conversation.Service {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := conversation.New(context.Background(), db, r)
	t.Cleanup(s.Close)
	return s
}

func TestAcceptedMessageAndReplyLifecycleAreLogged(t *testing.T) {
	logs := capture(t)
	var trace string
	s := open(t, scripted{trace: &trace})
	accepted, err := s.Accept("Call the SECRETPLUMBER")
	if err != nil {
		t.Fatal(err)
	}
	a := first(logs(), "message accepted")
	if a == nil || a["text_len"] != float64(len("Call the SECRETPLUMBER")) || a["reply_id"] != accepted.ReplyID || a["entry_id"] != accepted.Entry.ID {
		t.Fatalf("accept log incomplete: %v", a)
	}
	started := wait(t, logs, "reply started")
	done := wait(t, logs, "reply finished")
	if started["reply_id"] != accepted.ReplyID || done["reply_id"] != accepted.ReplyID || done["entries"] != float64(2) || done["kinds"] != "minder,preview" || done["duration_ms"] == nil {
		t.Fatalf("reply log incomplete: %v %v", started, done)
	}
	if trace != accepted.ReplyID {
		t.Fatalf("reply context trace %q, want %q", trace, accepted.ReplyID)
	}
	for _, l := range logs() {
		b, _ := json.Marshal(l)
		if strings.Contains(string(b), "SECRETPLUMBER") {
			t.Fatalf("message text reached the log: %s", b)
		}
	}
}

func TestRejectedMessagesAreLoggedWithReason(t *testing.T) {
	logs := capture(t)
	s := open(t, scripted{})
	if _, err := s.Accept("   "); err == nil {
		t.Fatal("empty accepted")
	}
	if r := first(logs(), "message rejected"); r == nil || r["reason"] != "length" || r["text_len"] != float64(0) {
		t.Fatalf("length rejection not logged: %v", r)
	}
}

func TestReplyErrorAndPanicAreLoggedWithCause(t *testing.T) {
	logs := capture(t)
	s := open(t, scripted{err: errors.New("provider exploded")})
	if _, err := s.Accept("one"); err != nil {
		t.Fatal(err)
	}
	failed := wait(t, logs, "reply failed")
	if failed["level"] != "ERROR" || failed["error"] != "provider exploded" || failed["error_code"] != "internal_error" {
		t.Fatalf("failure not logged: %v", failed)
	}
	logs2 := capture(t)
	p := open(t, scripted{panic: "kaboom"})
	if _, err := p.Accept("two"); err != nil {
		t.Fatal(err)
	}
	pan := wait(t, logs2, "reply panicked")
	if pan["panic"] != "kaboom" || !strings.Contains(pan["stack"].(string), "goroutine") {
		t.Fatalf("panic not logged: %v", pan)
	}
}

func TestBusyAndClosedAreLogged(t *testing.T) {
	logs := capture(t)
	release := make(chan struct{})
	var once sync.Once
	s := open(t, blocking{release})
	t.Cleanup(func() { once.Do(func() { close(release) }) }) // runs before Close, which waits for the reply
	if _, err := s.Accept("one"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Accept("two"); err == nil {
		t.Fatal("busy not enforced")
	}
	if r := first(logs(), "message rejected"); r == nil || r["reason"] != "busy" {
		t.Fatalf("busy rejection not logged: %v", r)
	}
}

type blocking struct{ release chan struct{} }

func (b blocking) Reply(context.Context, string) ([]domain.Entry, error) {
	<-b.release
	return nil, nil
}
