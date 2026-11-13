package capture

import (
	"context"
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"io"
	"strings"
	"unicode/utf8"
)

func (s *Service) Answer(ctx context.Context, id string, revision int, text string) (domain.Entry, error) {
	return s.answer(ctx, id, revision, text, false, "")
}
func (s *Service) Acknowledge(ctx context.Context, id string, revision int) (domain.Entry, error) {
	return s.answer(ctx, id, revision, "Continue with the readable content.", true, "")
}
func (s *Service) answer(ctx context.Context, id string, revision int, text string, ack bool, target string) (out domain.Entry, err error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) < 1 || utf8.RuneCountInString(text) > 4000 {
		return out, &domain.Invalid{Field: "answer", Message: "Use an answer with 1 to 4,000 characters."}
	}
	err = s.repository.Transact(func(tx port.Transaction) error {
		c, e := tx.Capture(id)
		if e != nil {
			return e
		}
		if c.Deleted || c.Revision != revision || c.Status == "processing" || c.Status == "received" || c.Status == "reviewed" {
			return domain.ErrConflict
		}
		if target != "" {
			found := false
			for _, candidate := range c.Candidates {
				if candidate.ID == target && (candidate.State == "pending" || candidate.State == "needs-answer") {
					found = true
				}
			}
			if !found {
				return domain.ErrConflict
			}
		}
		c.Answers = append(c.Answers, domain.CaptureAnswer{Revision: revision, CandidateID: target, Text: text})
		c.Revision++
		if ack {
			c.CoverageAcknowledged = true
		}
		for n := range c.Candidates {
			candidate := &c.Candidates[n]
			if candidate.State == "confirmed" || candidate.State == "discarded" || candidate.State == "superseded" || (target != "" && candidate.ID != target) || (target == "" && candidate.State == "pending") {
				continue
			}
			candidate.State = "superseded"
			if e = failPending(tx, candidate.ChangeID); e != nil {
				return e
			}
		}
		c.Status = "received"
		c.NextAt = 0
		c.LeaseUntil = 0
		c.Attempts = 0
		c.Error = ""
		if e = tx.PutCapture(c); e != nil {
			return e
		}
		out, e = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Answer received. Processing.", Capture: &c})
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
			if e = failPending(tx, candidate.ChangeID); e != nil {
				return e
			}
		}
		c.Status = "reviewed"
		c.Questions = nil
		if e = tx.PutCapture(c); e != nil {
			return e
		}
		out, e = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Nothing to do. The email is kept.", Capture: &c})
		return e
	})
	return
}

// failPending withdraws a still-pending preview so it can no longer be confirmed.
func failPending(tx port.Transaction, changeID string) error {
	if changeID == "" {
		return nil
	}
	p, e := tx.Pending(changeID)
	if e != nil || p.State != "pending" {
		return e
	}
	p.State = "failed"
	if e = tx.PutPending(p); e != nil {
		return e
	}
	entry, e := tx.Entry(p.EntryID)
	if e != nil {
		return e
	}
	entry.Change.State = "failed"
	_, e = tx.PutEntry(entry)
	return e
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
		if c.Calls >= 10 {
			return &domain.Invalid{Field: "capture", Message: "The automatic extraction limit is reached. Provide a shorter source."}
		}
		c.Status = "received"
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

// Reserved for typed non-transient extraction failures.
var ErrDocument = errors.New("The email could not be parsed. The original is available.")

func (s *Service) AnswerCandidate(ctx context.Context, id string, revision int, candidateID, text string) (domain.Entry, error) {
	return s.answer(ctx, id, revision, text, false, candidateID)
}
