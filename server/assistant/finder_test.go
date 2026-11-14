package assistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEveryLookupKindDescribesCostAndFields(t *testing.T) {
	want := map[LookupKind]string{LookupMaps: "postal code", LookupPlaces: "often missing", LookupSearch: "no structured address"}
	if len(LookupKinds) != len(want) {
		t.Fatalf("%d kinds", len(LookupKinds))
	}
	for _, info := range LookupKinds {
		if !strings.Contains(info.Description, "credit") || !strings.Contains(info.Description, want[info.Kind]) {
			t.Fatalf("%s: %q", info.Kind, info.Description)
		}
	}
}

func TestHitJSONOmitsEmptyFields(t *testing.T) {
	b, _ := json.Marshal(Hit{Title: "SnowWorld", Website: "https://example.test"})
	if string(b) != `{"title":"SnowWorld","website":"https://example.test"}` {
		t.Fatal(string(b))
	}
}
