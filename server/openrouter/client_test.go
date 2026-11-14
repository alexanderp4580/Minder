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

func TestRequestsAllowOneHundredThousandOutputTokens(t *testing.T) {
	var seen map[string]any
	m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"Hi"}}],"usage":{"cost":0.001}}`, &seen)
	if _, e := m.Complete(context.Background(), []assistant.Message{{Role: "user", Content: "Hello"}}); e != nil {
		t.Fatal(e)
	}
	if limit, _ := seen["max_tokens"].(float64); limit != 100000 {
		t.Fatalf("max_tokens %v, want 100000", seen["max_tokens"])
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
	for _, want := range []string{"reminder_offsets_minutes", `"options"`, `"maxItems":3`, `"summary"`, `"notification"`} {
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

func TestChatToolsIncludeTheFinishedItemsLookup(t *testing.T) {
	for _, tl := range tools() {
		fn := tl.(map[string]any)["function"].(map[string]any)
		if fn["name"] != "find_finished_items" {
			continue
		}
		props := fn["parameters"].(map[string]any)["properties"].(map[string]any)
		if _, ok := props["query"]; !ok {
			t.Fatal("no query property")
		}
		if _, ok := props["list"]; !ok {
			t.Fatal("no list property")
		}
		return
	}
	t.Fatal("find_finished_items is not offered")
}

func toolNames(t *testing.T, c *Client, messages []assistant.Message) []string {
	t.Helper()
	var seen map[string]any
	reply := `{"choices":[{"finish_reason":"stop","message":{"content":"Hi"}}],"usage":{"cost":0.001}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&seen)
		w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	c.URL = srv.URL
	if _, e := c.Complete(context.Background(), messages); e != nil {
		t.Fatal(e)
	}
	var names []string
	for _, tool := range seen["tools"].([]any) {
		names = append(names, tool.(map[string]any)["function"].(map[string]any)["name"].(string))
	}
	return names
}

func TestToolSetFollowsPurpose(t *testing.T) {
	has := func(purpose string) map[string]bool {
		set := map[string]bool{}
		for _, name := range toolNames(t, &Client{Key: "k"}, []assistant.Message{{Purpose: purpose, Role: "user", Content: "Hello"}}) {
			set[name] = true
		}
		return set
	}
	for purpose, want := range map[string]map[string]bool{
		assistant.PurposeChat:  {"propose_create": true, "offer_lookup": false},
		assistant.PurposeOffer: {"propose_create": true, "offer_lookup": true},
		assistant.PurposeEmail: {"extract_capture": true, "propose_create": false, "offer_lookup": false},
	} {
		got := has(purpose)
		for name, expected := range want {
			if got[name] != expected {
				t.Errorf("purpose %q: %s present=%v, want %v", purpose, name, got[name], expected)
			}
		}
	}
}

func TestOfferToolRequiresAKindAndAQueryAndListsEveryKind(t *testing.T) {
	function := offerTool().(map[string]any)["function"].(map[string]any)
	parameters := function["parameters"].(map[string]any)
	if required := parameters["required"].([]string); len(required) != 2 || required[0] != "kind" || required[1] != "query" {
		t.Fatalf("required %v", required)
	}
	kind := parameters["properties"].(map[string]any)["kind"].(map[string]any)
	enum := kind["enum"].([]string)
	if len(enum) != len(assistant.LookupKinds) {
		t.Fatalf("enum %v", enum)
	}
	for i, info := range assistant.LookupKinds {
		if enum[i] != string(info.Kind) || !strings.Contains(kind["description"].(string), info.Description) {
			t.Fatalf("kind %s is missing from the tool", info.Kind)
		}
	}
}

func TestItemToolsExposeDetailPairs(t *testing.T) {
	for _, purpose := range []string{assistant.PurposeChat, assistant.PurposeEmail} {
		var seen map[string]any
		m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"Hi"}}],"usage":{"cost":0.001}}`, &seen)
		if _, err := m.Complete(context.Background(), []assistant.Message{{Role: "system", Purpose: purpose, Content: "Review"}}); err != nil {
			t.Fatal(err)
		}
		fn := seen["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
		props := fn["parameters"].(map[string]any)["properties"].(map[string]any)
		if purpose == assistant.PurposeEmail {
			props = props["candidates"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["proposal"].(map[string]any)["properties"].(map[string]any)
		}
		d, ok := props["details"].(map[string]any)
		if !ok || d["type"] != "array" {
			t.Fatalf("%s has no detail list", purpose)
		}
		pair := d["items"].(map[string]any)["properties"].(map[string]any)
		if pair["label"] == nil || pair["value"] == nil || props["clear_details"] == nil {
			t.Fatal(props)
		}
	}
}
