package domain_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
)

func TestEveryFailureReasonHasAFixedSentence(t *testing.T) {
	for _, reason := range []domain.FailureReason{domain.ReasonCutOff, domain.ReasonEvidence, domain.ReasonNoReview, domain.ReasonUnreadable, domain.ReasonUnparsed, domain.ReasonLimit, domain.ReasonOther} {
		sentence := reason.Sentence()
		if !strings.HasPrefix(sentence, "I could not read this email: ") || !strings.HasSuffix(sentence, " The original is kept.") {
			t.Fatalf("%s: %q", reason, sentence)
		}
	}
}

func TestReasonOfReadsTheReasonOfAModelError(t *testing.T) {
	wrapped := fmt.Errorf("extract: %w", &domain.ModelError{Message: "cut", Reason: domain.ReasonCutOff})
	for err, want := range map[error]domain.FailureReason{
		&domain.ModelError{Message: "x", Reason: domain.ReasonEvidence}: domain.ReasonEvidence,
		wrapped:                         domain.ReasonCutOff,
		domain.ErrLimit:                 domain.ReasonLimit,
		&domain.ModelError{Message: "x"}: domain.ReasonOther,
		errors.New("disk"):              domain.ReasonOther,
	} {
		if got := domain.ReasonOf(err); got != want {
			t.Fatalf("%v: %s, want %s", err, got, want)
		}
	}
}
