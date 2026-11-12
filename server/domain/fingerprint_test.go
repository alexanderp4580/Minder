package domain

import (
	"context"
	"strings"
	"testing"
)

func TestFingerprintIdentifiesWithoutRevealing(t *testing.T) {
	a, b := Fingerprint("token-one"), Fingerprint("token-two")
	if len(a) != 8 || a == b || a != Fingerprint("token-one") || strings.Contains("token-one", a) {
		t.Fatalf("fingerprint unusable: %q %q", a, b)
	}
}

func TestTraceTravelsInTheContext(t *testing.T) {
	if TraceID(context.Background()) != "" {
		t.Fatal("empty context has a trace")
	}
	if got := TraceID(WithTrace(context.Background(), "abc")); got != "abc" {
		t.Fatalf("trace %q", got)
	}
}
