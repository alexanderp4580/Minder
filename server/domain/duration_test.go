package domain_test

import (
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
)

func TestParseDurationReadsTheQuickReplies(t *testing.T) {
	for text, want := range map[string]int{
		"30 minutes": 30, "45 minutes": 45, "1 hour": 60, "90 minutes": 90, "1 hour 30 minutes": 90,
		"2 hours": 120, " 1 HOUR ": 60, "1 hr": 60, "1h 30m": 90, "15 min": 15, "1 hour.": 60,
	} {
		if got, ok := domain.ParseDuration(text); !ok || got != want {
			t.Fatalf("%q: %d %v, want %d", text, got, ok, want)
		}
	}
}

func TestParseDurationRejectsEverythingElse(t *testing.T) {
	for _, text := range []string{"", "soon", "1 hour please", "about an hour", "0 minutes", "-5 minutes", "20000 minutes", "21 November", "1", "hour"} {
		if got, ok := domain.ParseDuration(text); ok {
			t.Fatalf("%q was read as %d", text, got)
		}
	}
}

func TestAsksDurationRecognisesDurationQuestions(t *testing.T) {
	for question, want := range map[string]bool{"How long will it take?": true, "What is the duration?": true, "how LONG is it": true, "What date?": false, "Where is it?": false, "": false} {
		if got := domain.AsksDuration(question); got != want {
			t.Fatalf("%q: %v", question, got)
		}
	}
}
