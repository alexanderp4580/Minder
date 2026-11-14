package assistant

import (
	"encoding/json"
	"testing"
)

func TestToolCallsAndResultsKeepTheirIdentityOnTheWire(t *testing.T) {
	raw, err := json.Marshal([]Message{
		{Role: "assistant", Content: "", ToolCalls: []Call{{ID: "c1", Name: "read_source", Arguments: `{"source_id":"s"}`}}},
		{Role: "tool", ToolCallID: "c1", Content: `{"ok":true}`},
		{Role: "user", Content: "Hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var wire []map[string]any
	json.Unmarshal(raw, &wire)
	call := wire[0]["tool_calls"].([]any)[0].(map[string]any)
	function := call["function"].(map[string]any)
	if call["id"] != "c1" || call["type"] != "function" || function["name"] != "read_source" || function["arguments"] != `{"source_id":"s"}` {
		t.Fatalf("%v", wire[0])
	}
	if wire[1]["role"] != "tool" || wire[1]["tool_call_id"] != "c1" || wire[1]["content"] != `{"ok":true}` {
		t.Fatalf("%v", wire[1])
	}
	if _, has := wire[2]["tool_calls"]; has || wire[2]["tool_call_id"] != nil {
		t.Fatalf("a plain message grew tool fields: %v", wire[2])
	}
}
