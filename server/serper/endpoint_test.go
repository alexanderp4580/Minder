package serper

import (
	"testing"

	"github.com/alexanderp4580/Minder/server/assistant"
)

func TestEveryLookupToolKindHasAnEndpoint(t *testing.T) {
	for _, tool := range assistant.LookupTools {
		if _, ok := endpoints[tool.Kind]; !ok {
			t.Fatalf("%s has no endpoint", tool.Kind)
		}
	}
}

func TestEndpointCreditsMatchTheToolDescriptions(t *testing.T) {
	want := map[assistant.LookupKind]int{assistant.LookupMaps: 3, assistant.LookupPlaces: 1, assistant.LookupSearch: 1}
	for kind, credits := range want {
		if endpoints[kind].credits != credits {
			t.Fatalf("%s costs %d", kind, endpoints[kind].credits)
		}
	}
}
