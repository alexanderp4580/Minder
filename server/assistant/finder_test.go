package assistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLookupToolKindMapsEachToolName(t *testing.T) {
	for name, want := range map[string]LookupKind{"lookup_maps": LookupMaps, "lookup_places": LookupPlaces, "lookup_search": LookupSearch} {
		if kind, ok := LookupToolKind(name); !ok || kind != want {
			t.Fatalf("%s -> %q %v", name, kind, ok)
		}
	}
	if _, ok := LookupToolKind("propose_create"); ok {
		t.Fatal("a proposal tool is not a lookup")
	}
}

func TestEveryLookupToolDescribesCostAndFields(t *testing.T) {
	want := map[LookupKind]string{LookupMaps: "postal code", LookupPlaces: "often missing", LookupSearch: "no structured address"}
	if len(LookupTools) != len(want) {
		t.Fatalf("%d tools", len(LookupTools))
	}
	for _, tool := range LookupTools {
		if !strings.Contains(tool.Description, "credit") || !strings.Contains(tool.Description, want[tool.Kind]) {
			t.Fatalf("%s: %q", tool.Name, tool.Description)
		}
	}
}

func TestHitJSONOmitsEmptyFields(t *testing.T) {
	b, _ := json.Marshal(Hit{Title: "SnowWorld", Website: "https://example.test"})
	if string(b) != `{"title":"SnowWorld","website":"https://example.test"}` {
		t.Fatal(string(b))
	}
}
