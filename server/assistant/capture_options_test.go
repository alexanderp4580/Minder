package assistant

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"reflect"
	"testing"
)

func TestDurationQuestionGetsSuggestionsWhenModelOmitsThem(t *testing.T) {
	extraction := domain.CaptureExtraction{Candidates: []domain.CaptureCandidate{{Question: "How long will the appointment take?"}}}
	normaliseExtraction(&extraction)
	if !reflect.DeepEqual(extraction.Candidates[0].Options, []string{"30 minutes", "45 minutes", "1 hour"}) {
		t.Fatal("duration suggestions missing", extraction.Candidates[0].Options)
	}
}
