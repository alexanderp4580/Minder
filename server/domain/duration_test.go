package domain_test

import (
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
)

func TestAsksDurationRecognisesDurationQuestions(t *testing.T) {
	for question, want := range map[string]bool{"How long will it take?": true, "What is the duration?": true, "how LONG is it": true, "What date?": false, "Where is it?": false, "": false} {
		if got := domain.AsksDuration(question); got != want {
			t.Fatalf("%q: %v", question, got)
		}
	}
}
