package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
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

func TestSuccessfulPushIsLoggedWithoutTheToken(t *testing.T) {
	logs := capture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"name":"projects/p/messages/42"}`)) }))
	defer server.Close()
	f := FCM{ProjectID: "test", Client: server.Client(), Endpoint: server.URL, AccessToken: func(context.Context) (string, error) { return "ACCESSSECRET", nil }}
	if _, e := f.Send(context.Background(), "DEVICETOKENSECRET", map[string]string{"reminder_id": "r1", "title": "PRIVATETITLE"}); e != nil {
		t.Fatal(e)
	}
	r := first(logs(), "push sent")
	if r == nil || r["status"] != float64(200) || r["message_name"] != "projects/p/messages/42" || r["token_fp"] == nil || r["reminder_id"] != "r1" || r["duration_ms"] == nil || r["project_id"] != "test" {
		t.Fatalf("send log incomplete: %v", r)
	}
	for _, l := range logs() {
		b, _ := json.Marshal(l)
		for _, secret := range []string{"DEVICETOKENSECRET", "ACCESSSECRET", "PRIVATETITLE"} {
			if strings.Contains(string(b), secret) {
				t.Fatalf("%s reached the log: %s", secret, b)
			}
		}
	}
}

func TestRejectedPushLogsFirebaseStatusAndErrorCode(t *testing.T) {
	logs := capture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":{"code":404,"status":"NOT_FOUND","message":"Requested entity was not found.","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`))
	}))
	defer server.Close()
	f := FCM{ProjectID: "test", Client: server.Client(), Endpoint: server.URL, AccessToken: func(context.Context) (string, error) { return "a", nil }}
	invalid, e := f.Send(context.Background(), "device", map[string]string{"reminder_id": "r1"})
	if e == nil || !invalid {
		t.Fatal("expected invalid token")
	}
	r := first(logs(), "push rejected")
	if r == nil || r["level"] != "WARN" || r["status"] != float64(404) || r["fcm_status"] != "NOT_FOUND" || r["fcm_error_code"] != "UNREGISTERED" || r["token_invalid"] != true || !strings.Contains(r["fcm_message"].(string), "not found") {
		t.Fatalf("rejection not logged: %v", r)
	}
}

func TestPushAuthorizationAndConnectionCausesAreLogged(t *testing.T) {
	logs := capture(t)
	f := FCM{ProjectID: "test", AccessToken: func(context.Context) (string, error) {
		return "", errors.New("oauth2: cannot fetch token: 400 Bad Request invalid_grant")
	}}
	if _, e := f.Send(context.Background(), "device", nil); e == nil {
		t.Fatal("expected failure")
	}
	r := first(logs(), "push failed")
	if r == nil || r["stage"] != "authorize" || !strings.Contains(r["error"].(string), "invalid_grant") {
		t.Fatalf("authorization cause not logged: %v", r)
	}
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := closed.URL
	closed.Close()
	f = FCM{ProjectID: "test", Endpoint: url, AccessToken: func(context.Context) (string, error) { return "a", nil }}
	f.Send(context.Background(), "device", nil)
	var connect map[string]any
	for _, l := range logs() {
		if l["msg"] == "push failed" && l["stage"] == "connect" {
			connect = l
		}
	}
	if connect == nil || !strings.Contains(connect["error"].(string), "connection refused") {
		t.Fatalf("connection cause not logged: %v", connect)
	}
}
