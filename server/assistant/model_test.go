package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func provider(t *testing.T, reply string, seen *map[string]any) *OpenRouter {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			json.NewDecoder(r.Body).Decode(seen)
		}
		w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return &OpenRouter{Key: "k", URL: srv.URL}
}

func TestReplyCutOffByTheTokenLimitIsAModelError(t *testing.T) {
	m := provider(t, `{"choices":[{"finish_reason":"length","message":{"content":null,"tool_calls":[{"id":"c","function":{"name":"propose_update","arguments":"{\"item_id\": \"i1\", \"expected_ver"}}]}}],"usage":{"cost":0.001}}`, nil)
	_, e := m.Complete(context.Background(), []Message{{Role: "user", Content: "Move it"}})
	var model *domain.ModelError
	if !errors.As(e, &model) {
		t.Fatalf("a truncated reply must be a model error, got %v", e)
	}
}

func TestRequestLeavesRoomForThinkingAndSeveralToolCalls(t *testing.T) {
	var seen map[string]any
	m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"Hi"}}],"usage":{"cost":0.001}}`, &seen)
	if _, e := m.Complete(context.Background(), []Message{{Role: "user", Content: "Hello"}}); e != nil {
		t.Fatal(e)
	}
	if limit, _ := seen["max_tokens"].(float64); limit < 4000 {
		t.Fatalf("max_tokens %v leaves too little room", seen["max_tokens"])
	}
}
