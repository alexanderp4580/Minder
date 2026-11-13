package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/domain"
)

func provider(t *testing.T, reply string, seen *map[string]any) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			json.NewDecoder(r.Body).Decode(seen)
		}
		w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return &Client{Key: "k", URL: srv.URL}
}

func TestReplyCutOffByTheTokenLimitIsAModelError(t *testing.T) {
	m := provider(t, `{"choices":[{"finish_reason":"length","message":{"content":null,"tool_calls":[{"id":"c","function":{"name":"propose_update","arguments":"{\"item_id\": \"i1\", \"expected_ver"}}]}}],"usage":{"cost":0.001}}`, nil)
	_, e := m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "Move it"}})
	var model *domain.ModelError
	if !errors.As(e, &model) {
		t.Fatalf("a truncated reply must be a model error, got %v", e)
	}
}

func TestRequestLeavesRoomForThinkingAndSeveralToolCalls(t *testing.T) {
	var seen map[string]any
	m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"Hi"}}],"usage":{"cost":0.001}}`, &seen)
	if _, e := m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "Hello"}}); e != nil {
		t.Fatal(e)
	}
	if limit, _ := seen["max_tokens"].(float64); limit < 4000 {
		t.Fatalf("max_tokens %v leaves too little room", seen["max_tokens"])
	}
}

func TestImagePartsPreserveTextCaller(t *testing.T) {
	var seen map[string]any
	m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"Read"}}],"usage":{"cost":0.001}}`, &seen)
	_, e := m.Complete(context.Background(), []assistant.Message{{Role: "user", Parts: []assistant.ContentPart{{Type: "text", Text: "Read attachment"}, {Type: "image_url", ImageURL: &assistant.ImageURL{URL: "data:image/png;base64,AA=="}}}}})
	if e != nil {
		t.Fatal(e)
	}
	messages := seen["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	if len(content) != 2 || content[1].(map[string]any)["image_url"].(map[string]any)["url"] != "data:image/png;base64,AA==" {
		t.Fatal(content)
	}
}

func TestExtractionSchemaAllowsOptionsAndOffsets(t *testing.T) {
	raw, err := json.Marshal(captureTools())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"reminder_offsets_minutes", `"options"`, `"maxItems":3`} {
		if !strings.Contains(string(raw), want) {
			t.Fatal("extract_capture schema lacks", want)
		}
	}
}

func TestChatToolsIncludeCancelAndDelete(t *testing.T) {
	names := map[string][]string{}
	for _, tl := range tools() {
		fn := tl.(map[string]any)["function"].(map[string]any)
		req := fn["parameters"].(map[string]any)["required"].([]string)
		names[fn["name"].(string)] = req
	}
	for _, n := range []string{"propose_cancel", "propose_delete"} {
		req, ok := names[n]
		if !ok || len(req) != 2 || req[0] != "item_id" || req[1] != "expected_version" {
			t.Fatalf("%s: %v %v", n, ok, req)
		}
	}
}
