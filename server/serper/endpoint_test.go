package serper

import (
	"testing"

	"github.com/alexanderp4580/Minder/server/assistant"
)

func TestEveryLookupKindHasAnEndpoint(t *testing.T) {
	for _, info := range assistant.LookupKinds {
		if _, ok := endpoints[info.Kind]; !ok {
			t.Fatalf("%s has no endpoint", info.Kind)
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
