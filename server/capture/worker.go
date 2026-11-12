package capture

import (
	"bytes"
	"context"
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"log/slog"
	"time"
)

func (s *Service) Run(ctx context.Context) {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		if e := s.cleanupDeleted(); e != nil {
			slog.Warn("email source cleanup failed")
		}
		if e := s.deliverNotifications(ctx, time.Now()); e != nil {
			slog.Warn("email review delivery failed")
		}
		if e := s.ProcessNext(ctx, time.Now()); e != nil && !errors.Is(e, context.Canceled) {
			slog.Warn("email processing failed", "error_code", domain.ErrorCode(e))
		}
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}
func (s *Service) ProcessNext(ctx context.Context, now time.Time) error {
	c, e := s.repository.ClaimCapture(now)
	if errors.Is(e, domain.ErrNotFound) {
		return nil
	}
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if s.Parser == nil || s.extractor == nil || s.Preview == nil {
		return s.fail(c, now, ErrDocument, true)
	}
	f, e := s.files.Open(c.ID, "original")
	if e != nil {
		return s.fail(c, now, e, false)
	}
	document, e := s.Parser.Parse(ctx, f)
	f.Close()
	if e != nil {
		return s.fail(c, now, ErrDocument, true)
	}
	for _, part := range document.Parts {
		if part.Attachment {
			if _, _, e = s.files.Save(ctx, c.ID, part.ID, bytes.NewReader(part.Data), 10<<20); e != nil {
				return s.fail(c, now, e, false)
			}
		}
	}
	document.CaptureID = c.ID
	document.PreviousCandidates = c.Candidates
	extraction, e := s.extractor.Extract(ctx, document, c.Answers)
	if e != nil {
		return s.fail(c, now, e, errors.Is(e, domain.ErrLimit))
	}
	e = s.repository.Transact(func(tx port.Transaction) error {
		current, e := tx.Capture(c.ID)
		if e != nil {
			return e
		}
		if current.Deleted || current.Revision != c.Revision || current.LeaseUntil != c.LeaseUntil {
			return domain.ErrConflict
		}
		current.Subject = document.Subject
		current.OriginalDate = document.OriginalDate
		current.Attachments = append(current.Attachments[:1], document.Attachments...)
		current.Coverage = document.Coverage
		current.Questions = extraction.Questions
		current.LeaseUntil = 0
		current.NextAt = 0
		current.Error = ""
		// Terminal candidates remain authoritative when answers regenerate proposals.
		terminal := map[string]bool{}
		for _, candidate := range current.Candidates {
			if candidate.State == "confirmed" || candidate.State == "discarded" {
				terminal[candidate.ID] = true
			}
		}
		for n, candidate := range extraction.Candidates {
			if candidate.ID == "" {
				candidate.ID = stableCandidateID(candidate, n)
			}
			if terminal[candidate.ID] {
				continue
			}
			existing := false
			for _, previous := range current.Candidates {
				if previous.ID == candidate.ID && previous.State != "superseded" {
					existing = true
				}
			}
			if existing {
				continue
			}
			if len(document.PreviousCandidates) > 0 {
				known := false
				for _, previous := range document.PreviousCandidates {
					if previous.ID == candidate.ID {
						known = true
					}
				}
				if !known {
					return &domain.Invalid{Field: "capture", Message: "The model changed candidate identities. Retry the review."}
				}
			}
			candidate.Revision = current.Revision
			candidate.State = "pending"
			if candidate.Question != "" {
				candidate.State = "needs-answer"
			}
			candidate.Proposal.CaptureID = current.ID
			candidate.Proposal.CandidateID = candidate.ID
			candidate.Proposal.CaptureRevision = candidate.Revision
			if candidate.Question == "" {
				entry, e := s.Preview(tx, candidate.Proposal)
				if e != nil {
					return e
				}
				candidate.ChangeID = entry.Change.ID
			}
			current.Candidates = append(current.Candidates, candidate)
		}
		current.UpdateReviewState()
		if current.Status == "needs-answer" || current.Status == "ready-for-review" {
			if e = tx.QueueCaptureNotifications(current.ID, current.Revision); e != nil {
				return e
			}
		}
		if e = tx.PutCapture(current); e != nil {
			return e
		}
		_, e = tx.PutEntry(domain.Entry{ID: current.EntryID, Kind: "capture", Text: "Review the email details.", Capture: &current})
		return e
	})
	if e != nil {
		return s.fail(c, now, e, false)
	}
	return nil
}
func (s *Service) fail(c domain.Capture, now time.Time, cause error, permanent bool) error {
	delays := []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute}
	err := s.repository.Transact(func(tx port.Transaction) error {
		current, e := tx.Capture(c.ID)
		if e != nil {
			return e
		}
		if current.Deleted || current.Revision != c.Revision || current.LeaseUntil != c.LeaseUntil {
			return domain.ErrConflict
		}
		current.Attempts++
		current.LeaseUntil = 0
		current.Error = "Email processing failed. The original is available."
		if errors.Is(cause, domain.ErrLimit) {
			current.Error = "The monthly model limit is reached. Retry when budget is available."
		}
		var invalid *domain.Invalid
		if errors.As(cause, &invalid) {
			current.Error = invalid.Message
			permanent = true
		}
		current.Status = "failed"
		if !permanent && current.Attempts <= len(delays) {
			current.Status = "received"
			current.NextAt = now.Add(delays[current.Attempts-1]).Unix()
		}
		if e = tx.PutCapture(current); e != nil {
			return e
		}
		_, e = tx.PutEntry(domain.Entry{ID: current.EntryID, Kind: "capture", Text: current.Error, Capture: &current})
		return e
	})
	if err != nil {
		return err
	}
	return cause
}

func (s *Service) cleanupDeleted() error {
	repository, ok := s.repository.(interface {
		DeletedCaptures() ([]domain.Capture, error)
	})
	if !ok {
		return nil
	}
	sources, e := repository.DeletedCaptures()
	if e != nil {
		return e
	}
	for _, source := range sources {
		if e = s.files.DeleteSource(source.ID); e != nil {
			return e
		}
		if e = s.repository.Transact(func(tx port.Transaction) error {
			c, e := tx.Capture(source.ID)
			if e != nil {
				return e
			}
			c.FilesDeleted = true
			return tx.PutCapture(c)
		}); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) deliverNotifications(ctx context.Context, now time.Time) error {
	if s.Sender == nil {
		return nil
	}
	repo, ok := s.repository.(interface {
		CaptureDeliveries(time.Time) ([]domain.CaptureDelivery, error)
		FinishCaptureDelivery(domain.CaptureDelivery, bool, time.Time) error
	})
	if !ok {
		return nil
	}
	deliveries, e := repo.CaptureDeliveries(now)
	if e != nil {
		return e
	}
	for _, delivery := range deliveries {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		invalid, sendErr := s.Sender.Send(ctx, delivery.Token, map[string]string{"capture_id": delivery.CaptureID, "type": "capture_review", "title": "Minder", "body": "An email needs your review."})
		cancel()
		next := time.Time{}
		if sendErr != nil && !invalid {
			next = now.Add(time.Minute)
		}
		if e = repo.FinishCaptureDelivery(delivery, invalid, next); e != nil {
			return e
		}
	}
	return nil
}
