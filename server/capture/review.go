package capture

import (
	"context"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/review"
	"io"
)

// Acknowledge accepts that unreadable parts stay unread, so the review can finish.
func (s *Service) Acknowledge(_ context.Context, id string, revision int) (out domain.Entry, err error) {
	err = s.repository.Transact(func(tx port.Transaction) error {
		c, e := tx.Capture(id)
		if e != nil {
			return e
		}
		if c.Deleted || c.Revision != revision || c.Status != "needs-answer" {
			return domain.ErrConflict
		}
		c.CoverageAcknowledged = true
		c.UpdateReviewState()
		if e = tx.PutCapture(c); e != nil {
			return e
		}
		out, e = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Review the email details.", Capture: &c})
		return e
	})
	return
}

// Dismiss closes an open review without changing items. The source stays available.
func (s *Service) Dismiss(ctx context.Context, id string, revision int) (out domain.Entry, err error) {
	err = s.repository.Transact(func(tx port.Transaction) error {
		c, e := tx.Capture(id)
		if e != nil {
			return e
		}
		if c.Deleted || c.Revision != revision || c.Status == "processing" || c.Status == "received" || c.Status == "reviewed" || c.Status == "failed" {
			return domain.ErrConflict
		}
		for n := range c.Candidates {
			candidate := &c.Candidates[n]
			if candidate.State == "confirmed" || candidate.State == "discarded" {
				continue
			}
			candidate.State = "discarded"
			if e = review.WithdrawPreview(tx, candidate.ChangeID); e != nil {
				return e
			}
		}
		c.Status = "reviewed"
		c.Questions = nil
		if e = review.CloseQuestions(tx, c, "Questions dismissed."); e != nil {
			return e
		}
		if e = tx.PutCapture(c); e != nil {
			return e
		}
		out, e = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Nothing to do. The email is kept.", Capture: &c})
		return e
	})
	return
}

func (s *Service) Retry(id string) (out domain.Entry, err error) {
	err = s.repository.Transact(func(tx port.Transaction) error {
		c, e := tx.Capture(id)
		if e != nil {
			return e
		}
		if c.Deleted || c.Status != "failed" {
			return domain.ErrConflict
		}
		c.Revision++ // a new revision, so its push is not mistaken for the failure's
		c.Status = "received"
		c.Prepared = false
		c.Attempts = 0
		c.NextAt = 0
		c.LeaseUntil = 0
		c.Error = ""
		if e = tx.PutCapture(c); e != nil {
			return e
		}
		out, e = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Retry queued.", Capture: &c})
		return e
	})
	return
}
func (s *Service) Attachment(id, attachmentID string) (io.ReadCloser, domain.CaptureAttachment, error) {
	c, e := s.repository.Capture(id)
	if e != nil {
		return nil, domain.CaptureAttachment{}, e
	}
	if c.Deleted {
		return nil, domain.CaptureAttachment{}, domain.ErrNotFound
	}
	for _, a := range c.Attachments {
		if a.ID == attachmentID {
			f, e := s.files.Open(id, attachmentID)
			return f, a, e
		}
	}
	return nil, domain.CaptureAttachment{}, domain.ErrNotFound
}
