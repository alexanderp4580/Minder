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

func TestReviewSchemaAllowsOptionsOffsetsAndTargetedUpdates(t *testing.T) {
	raw, err := json.Marshal(sourceTools())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"reminder_offsets_minutes", `"options"`, `"maxItems":3`, `"summary"`, `"notification"`, `"expected_revision"`, `"revision"`, `"evidence"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatal("propose_review schema lacks", want)
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
		if !ok || len(req) != 1 || req[0] != "item_id" {
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
	// One tool family serves every request; only the lookup offer depends on the purpose.
	everything := func(offer bool) map[string]bool {
		return map[string]bool{"propose_create": true, "list_items": true, "find_finished_items": true, "find_sources": true, "read_source": true, "inspect_review": true, "propose_review": true, "offer_lookup": offer, "extract_capture": false}
	}
	for purpose, want := range map[string]map[string]bool{
		assistant.PurposeChat:  everything(false),
		assistant.PurposeOffer: everything(true),
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

func TestItemToolsExposeDetailPairsInChatAndInReviews(t *testing.T) {
	chat := tools()[0].(map[string]any)["function"].(map[string]any)["parameters"].(map[string]any)["properties"].(map[string]any)
	review := sourceTools()[3].(map[string]any)["function"].(map[string]any)["parameters"].(map[string]any)["properties"].(map[string]any)
	proposal := review["candidates"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["proposal"].(map[string]any)["properties"].(map[string]any)
	for name, props := range map[string]map[string]any{"chat": chat, "review": proposal} {
		d, ok := props["details"].(map[string]any)
		if !ok || d["type"] != "array" {
			t.Fatalf("%s has no detail list", name)
		}
		pair := d["items"].(map[string]any)["properties"].(map[string]any)
		if pair["label"] == nil || pair["value"] == nil || props["clear_details"] == nil {
			t.Fatal(props)
		}
	}
}

func TestAToolCallExchangeReachesTheProviderWithItsIdentity(t *testing.T) {
	var seen map[string]any
	m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"Done"}}],"usage":{"cost":0.001}}`, &seen)
	_, err := m.Complete(context.Background(), []assistant.Message{
		{Role: "user", Content: "What should I bring?"},
		{Role: "assistant", ToolCalls: []assistant.Call{{ID: "call-1", Name: "read_source", Arguments: `{"source_id":"mail"}`}}},
		{Role: "tool", ToolCallID: "call-1", Content: `{"parts":[]}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	messages := seen["messages"].([]any)
	call := messages[1].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)
	if call["id"] != "call-1" || messages[2].(map[string]any)["tool_call_id"] != "call-1" || messages[2].(map[string]any)["role"] != "tool" {
		t.Fatalf("%v", messages)
	}
}

func TestItemToolsLetTheAgentStartAndReopenTasks(t *testing.T) {
	fn := tools()[0].(map[string]any)["function"].(map[string]any)
	status, ok := fn["parameters"].(map[string]any)["properties"].(map[string]any)["status"].(map[string]any)
	if !ok {
		t.Fatal("status missing from", fn["name"])
	}
	enum := status["enum"].([]string)
	if len(enum) != 2 || enum[0] != "open" || enum[1] != "in_progress" {
		t.Fatal(enum)
	}
	reviews, _ := json.Marshal(sourceTools())
	if !strings.Contains(string(reviews), `"status":{`) {
		t.Fatal("review proposals lack status")
	}
}

func TestRequestsMarkCacheBreakpointsOnlyWhereTheCallerAsks(t *testing.T) {
	var seen map[string]any
	m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"Hi"}}],"usage":{"cost":0.001,"prompt_tokens":900,"prompt_tokens_details":{"cached_tokens":800}}}`, &seen)
	system := "RULES. Current local time: now."
	messages := []assistant.Message{
		{Role: "system", Content: system, CachePrefix: len("RULES.")},
		{Role: "user", Content: "Earlier"},
		{Role: "assistant", Content: "Reply"},
		{Role: "user", Content: "Hello", Cache: true},
	}
	if _, e := m.Complete(context.Background(), messages); e != nil {
		t.Fatal(e)
	}
	wire := seen["messages"].([]any)
	parts := func(i int) []any { p, _ := wire[i].(map[string]any)["content"].([]any); return p }
	sys := parts(0)
	if len(sys) != 2 {
		t.Fatalf("system content: %v", wire[0])
	}
	first, second := sys[0].(map[string]any), sys[1].(map[string]any)
	if first["text"] != "RULES." || first["cache_control"].(map[string]any)["type"] != "ephemeral" {
		t.Fatalf("stable block: %v", first)
	}
	if second["text"] != " Current local time: now." || second["cache_control"] != nil {
		t.Fatalf("dynamic block: %v", second)
	}
	if _, plain := wire[1].(map[string]any)["content"].(string); !plain {
		t.Fatalf("an unmarked message must stay a string: %v", wire[1])
	}
	last := parts(3)
	if len(last) != 1 || last[0].(map[string]any)["text"] != "Hello" || last[0].(map[string]any)["cache_control"].(map[string]any)["type"] != "ephemeral" {
		t.Fatalf("breakpoint message: %v", wire[3])
	}
}

func TestMessagesWithoutBreakpointsAreUnchanged(t *testing.T) {
	var seen map[string]any
	m := provider(t, `{"choices":[{"finish_reason":"stop","message":{"content":"Hi"}}],"usage":{"cost":0.001}}`, &seen)
	if _, e := m.Complete(context.Background(), []assistant.Message{{Role: "system", Content: "Rules."}, {Role: "user", Content: "Hello"}}); e != nil {
		t.Fatal(e)
	}
	for _, msg := range seen["messages"].([]any) {
		if _, plain := msg.(map[string]any)["content"].(string); !plain {
			t.Fatalf("content must stay a string without a breakpoint: %v", msg)
		}
	}
}
