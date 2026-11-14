package assistant

import (
	"strings"

	"github.com/alexanderp4580/Minder/server/domain"
)

// durationOptions are the quick replies for a duration question that came with none.
var durationOptions = []string{"30 minutes", "45 minutes", "1 hour"}

// normalizeReview caps quick replies at three, fills the usual duration choices and keeps only
// allowed reminder offsets, on the review that checkReview accepted.
func (l *loop) normalizeReview(patch *domain.ReviewPatch) {
	for n := range patch.Candidates {
		candidate := &patch.Candidates[n]
		candidate.Options = cleanOptions(candidate.Options)
		if domain.AsksDuration(candidate.Question) && len(candidate.Options) == 0 {
			candidate.Options = append([]string(nil), durationOptions...)
		}
		var allowed []int
		for _, minutes := range candidate.Proposal.ReminderOffsets {
			if _, err := domain.NormalizeOffsets([]int{minutes}); err == nil {
				allowed = append(allowed, minutes)
			}
		}
		candidate.Proposal.ReminderOffsets, _ = domain.NormalizeOffsets(allowed)
	}
}

func cleanOptions(in []string) []string {
	var out []string
	for _, option := range in {
		if option = strings.TrimSpace(option); option != "" && len(out) < 3 {
			out = append(out, option)
		}
	}
	return out
}
