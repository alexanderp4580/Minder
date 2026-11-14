package capture

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/review"
)

// applyDuration answers a "how long" question of the target candidate without asking the model.
// It reports false, changing nothing, when the question or the answer is not about a duration.
func (s *Service) applyDuration(tx port.Transaction, c *domain.Capture, target, text string) (domain.Entry, bool, error) {
	minutes, ok := domain.ParseDuration(text)
	index := -1
	for n, candidate := range c.Candidates {
		if candidate.ID == target && candidate.State == "needs-answer" && domain.AsksDuration(candidate.Question) {
			index = n
		}
	}
	if !ok || index < 0 {
		return domain.Entry{}, false, nil
	}
	c.Answers = append(c.Answers, domain.CaptureAnswer{Revision: c.Revision, CandidateID: target, Text: text})
	c.Revision++
	candidate := &c.Candidates[index]
	candidate.Proposal.DurationMinutes = minutes
	candidate.Question, candidate.Options = "", nil
	candidate.State, candidate.Revision = "queued", c.Revision
	candidate.Proposal.CaptureRevision = c.Revision
	if err := review.CloseQuestions(tx, *c, "Questions answered."); err != nil {
		return domain.Entry{}, false, err
	}
	if err := review.Advance(tx, c, s.Preview); err != nil {
		return domain.Entry{}, false, err
	}
	if review.HasOpenQuestions(*c) {
		if err := review.PostQuestions(tx, *c); err != nil {
			return domain.Entry{}, false, err
		}
	}
	c.UpdateReviewState()
	if err := tx.PutCapture(*c); err != nil {
		return domain.Entry{}, false, err
	}
	entry, err := tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Answer received.", Capture: c})
	return entry, err == nil, err
}
