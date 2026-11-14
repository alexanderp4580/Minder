package assistant

import (
	"strings"
	"testing"
)

func TestThePromptSaysOnlineMeetingsHaveNoLocation(t *testing.T) {
	for _, want := range []string{"Online meetings", "leave location empty", "join link", "website", "Online meeting ("} {
		if !strings.Contains(captureInstructions, want) {
			t.Fatalf("the capture prompt lacks %q", want)
		}
	}
}
