package serper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func transportFor(t *testing.T, handler http.HandlerFunc) transport {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return transport{key: "serper-secret", base: srv.URL}
}

func TestPostSendsKeyAndJSONPayload(t *testing.T) {
	var method, key, path string
	var payload map[string]any
	tr := transportFor(t, func(w http.ResponseWriter, r *http.Request) {
		method, key, path = r.Method, r.Header.Get("X-API-KEY"), r.URL.Path
		json.NewDecoder(r.Body).Decode(&payload)
		w.Write([]byte(`{"ok":true}`))
	})
	body, status, err := tr.post(context.Background(), "/maps", map[string]any{"q": "SnowWorld"})
	if err != nil || status != 200 || string(body) != `{"ok":true}` {
		t.Fatalf("%s %d %v", body, status, err)
	}
	if method != "POST" || key != "serper-secret" || path != "/maps" || payload["q"] != "SnowWorld" {
		t.Fatalf("%s %s %s %v", method, key, path, payload)
	}
}

func TestPostErrorsNeverCarryTheKeyOrBody(t *testing.T) {
	for _, status := range []int{401, 500} {
		tr := transportFor(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte("provider body mentions serper-secret"))
		})
		_, got, err := tr.post(context.Background(), "/maps", nil)
		if err == nil || got != status || strings.Contains(err.Error(), "serper-secret") || strings.Contains(err.Error(), "provider body") {
			t.Fatalf("%d: %d %v", status, got, err)
		}
	}
}

func TestPostRejectsABodyOverTheLimit(t *testing.T) {
	tr := transportFor(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(strings.Repeat("x", maxBody+1))) })
	if _, _, err := tr.post(context.Background(), "/maps", nil); err == nil {
		t.Fatal("a body over the limit was accepted")
	}
}

func TestPostWithoutAKeyMakesNoRequest(t *testing.T) {
	called := false
	tr := transportFor(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	tr.key = ""
	if _, _, err := tr.post(context.Background(), "/maps", nil); err == nil || called {
		t.Fatalf("err %v called %v", err, called)
	}
}

func TestPostStopsWhenTheContextEnds(t *testing.T) {
	tr := transportFor(t, func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		<-r.Context().Done()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, _, err := tr.post(ctx, "/maps", nil); err == nil || time.Since(started) > 2*time.Second {
		t.Fatalf("err %v after %v", err, time.Since(started))
	}
}
