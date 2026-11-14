package command

import (
	"errors"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/review"
)

// reviewCapture loads the capture a patch targets. A pasted-text source has none until its
// first publication, which creates the capture and the source record.
func (w *turnWriter) reviewCapture(patch domain.ReviewPatch) (domain.Capture, error) {
	ref, err := w.tx.Source(patch.SourceID)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		entry, e := w.tx.Entry(patch.SourceID)
		if e != nil || entry.Kind != "user" {
			return domain.Capture{}, domain.ErrConflict
		}
		ref = domain.SourceRef{ID: entry.ID, Kind: domain.SourceChat, EntryID: entry.ID, Label: domain.ChatLabel(entry.Text), CreatedAt: entry.CreatedAt}
	case err != nil:
		return domain.Capture{}, err
	}
	if ref.Deleted {
		return domain.Capture{}, domain.ErrConflict
	}
	c, err := w.tx.Capture(patch.SourceID)
	switch {
	case err == nil:
		if c.Deleted || c.Revision != patch.ExpectedRevision {
			return c, domain.ErrConflict
		}
		return c, nil
	case !errors.Is(err, domain.ErrNotFound):
		return c, err
	case ref.Kind != domain.SourceChat || patch.ExpectedRevision != 0:
		return c, domain.ErrConflict
	}
	ref.CaptureID = ref.ID
	if err = w.tx.PutSource(ref); err != nil {
		return c, err
	}
	return domain.Capture{ID: ref.ID, Kind: domain.SourceChat, Status: "processing", Revision: 1, Subject: ref.Label, ReceivedAt: ref.CreatedAt, EntryID: domain.NewID(),
		Attachments: []domain.CaptureAttachment{}, Candidates: []domain.CaptureCandidate{},
		Coverage: []domain.CaptureCoverage{{Part: "text", Status: "parsed"}}}, nil
}

// applyReview adds the patch's new candidates and replaces the candidates it updates. Candidates
// the patch leaves out, and confirmed or discarded ones, stay as they are. Any failure rolls the
// whole turn back.
func (w *turnWriter) applyReview(patch domain.ReviewPatch) error {
	c, err := w.reviewCapture(patch)
	if err != nil {
		return err
	}
	first := len(c.Candidates) == 0
	slots := map[string]int{}
	for n, candidate := range c.Candidates {
		slots[candidate.ID] = n
	}
	updates := 0
	for _, candidate := range patch.Candidates {
		if candidate.ID != "" {
			updates++
		}
	}
	if updates > 0 {
		c.Revision++ // an answer or correction is a new revision of the review
	}
	closeQuestions, seen := false, map[string]bool{}
	for _, candidate := range patch.Candidates {
		if candidate.ID == "" {
			candidate.ID = domain.NewID()
			c.Candidates = append(c.Candidates, bindCandidate(candidate, c))
			continue
		}
		slot, ok := slots[candidate.ID]
		if !ok || seen[candidate.ID] {
			return domain.ErrConflict
		}
		seen[candidate.ID] = true
		old := c.Candidates[slot]
		if old.State == "confirmed" || old.State == "discarded" || candidate.Revision != old.Revision {
			return domain.ErrConflict
		}
		if err = review.WithdrawPreview(w.tx, old.ChangeID); err != nil {
			return err
		}
		closeQuestions = closeQuestions || old.State == "needs-answer"
		if len(candidate.Evidence) == 0 {
			candidate.Evidence = old.Evidence
		}
		c.Candidates[slot] = bindCandidate(candidate, c)
	}
	if closeQuestions {
		if err = review.CloseQuestions(w.tx, c, "Questions answered."); err != nil {
			return err
		}
	}
	if patch.Summary != "" {
		if _, err = w.put(domain.Entry{Kind: "minder", Text: patch.Summary}); err != nil {
			return err
		}
	}
	if first {
		c.Notification = patch.Notification
	}
	if err = review.Advance(w.tx, &c, PreviewIn); err != nil {
		return err
	}
	c.UpdateReviewState()
	if err = w.tx.PutCapture(c); err != nil {
		return err
	}
	if first && len(c.Candidates) > 0 {
		// Only the first publication is news. The owner's own answers get no push.
		if err = w.tx.QueueCaptureNotifications(c.ID, c.Revision); err != nil {
			return err
		}
	}
	card, err := w.tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Review the email details.", Capture: &c})
	if err != nil {
		return err
	}
	w.entries = append(w.entries, card)
	if review.HasOpenQuestions(c) {
		return review.PostQuestions(w.tx, c)
	}
	return nil
}

// bindCandidate queues a candidate at the review's revision and ties its proposal to it.
func bindCandidate(candidate domain.CaptureCandidate, c domain.Capture) domain.CaptureCandidate {
	candidate.Revision, candidate.State, candidate.ChangeID = c.Revision, "queued", ""
	candidate.Proposal.CaptureID = c.ID
	candidate.Proposal.CandidateID = candidate.ID
	candidate.Proposal.CaptureRevision = candidate.Revision
	return candidate
}
