package capture

import (
	"fmt"
	"bytes"
	"context"
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/review"
	"log/slog"
	"time"
)

func (s *Service) Run(ctx context.Context) {
	if repository, ok := s.repository.(interface {
		CaptureIDs() (map[string]bool, error)
	}); ok {
		if files, ok := s.files.(interface {
			Cleanup(map[string]bool, time.Time) error
		}); ok {
			known, err := repository.CaptureIDs()
			if err == nil {
				err = files.Cleanup(known, time.Now())
			}
			if err != nil {
				slog.Warn("email orphan cleanup failed")
			}
		}
	}

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
	parent := ctx
	if err := parent.Err(); err != nil {
		return err
	}
	c, e := s.repository.ClaimCapture(now)
	if errors.Is(e, domain.ErrNotFound) {
		return nil
	}
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	failed := func(cause error) error {
		if parent.Err() != nil {
			if err := s.releaseClaim(c); err != nil {
				return err
			}
			return parent.Err()
		}
		return s.fail(c, cause)
	}
	if s.Parser == nil || s.extractor == nil || s.Preview == nil {
		return failed(ErrDocument)
	}
	f, e := s.files.Open(c.ID, "original")
	if e != nil {
		return failed(e)
	}
	document, e := s.Parser.Parse(ctx, f)
	f.Close()
	if e != nil {
		return failed(ErrDocument)
	}
	e = s.repository.Transact(func(tx port.Transaction) error {
		current, err := tx.Capture(c.ID)
		if err != nil {
			return err
		}
		if current.Deleted || current.Revision != c.Revision || current.LeaseUntil != c.LeaseUntil {
			return domain.ErrConflict
		}
		// Serialize file publication against deletion so cleanup cannot race a later write.
		for _, part := range document.Parts {
			if part.Attachment {
				if _, _, err = s.files.Save(ctx, c.ID, part.ID, bytes.NewReader(part.Data), 10<<20); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if e != nil {
		return failed(e)
	}

	document.CaptureID = c.ID
	document.PreviousCandidates = c.Candidates
	extraction, e := s.extractor.Extract(ctx, document, c.Answers)
	if e != nil {
		return failed(e)
	}
	slog.Info("email extraction completed", "source_id", c.ID, "revision", c.Revision, "candidates", len(extraction.Candidates))
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
		current.Unverified = extraction.Dropped
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
		previousByID := map[string]domain.CaptureCandidate{}
		for _, candidate := range current.Candidates {
			previousByID[candidate.ID] = candidate
		}
		first := len(current.Candidates) == 0
		returned := map[string]bool{}
		for n, candidate := range extraction.Candidates {
			id := candidate.ID
			if id == "" {
				id = stableCandidateID(candidate, n)
			}
			returned[id] = true
		}
		for id, candidate := range previousByID {
			if candidate.State == "superseded" && !returned[id] {
				return &domain.ModelError{Message: "The model omitted an item awaiting clarification."}
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
			candidate.State = "queued"
			candidate.Proposal.CaptureID = current.ID
			candidate.Proposal.CandidateID = candidate.ID
			candidate.Proposal.CaptureRevision = candidate.Revision
			if slot := supersededSlot(current.Candidates, candidate.ID); slot >= 0 {
				current.Candidates[slot] = candidate // a regenerated candidate keeps its place in the review order
			} else {
				current.Candidates = append(current.Candidates, candidate)
			}
		}
		if first && len(current.Candidates) > 0 {
			opening := review.Opening(document.Subject, current.Candidates)
			if current.Unverified > 0 {
				opening += fmt.Sprintf(" I could not verify %d more from this email.", current.Unverified)
			}
			if _, e = tx.PutEntry(domain.Entry{Kind: "minder", Text: opening}); e != nil {
				return e
			}
		}
		if e = review.Advance(tx, &current, s.Preview); e != nil {
			return e
		}
		current.UpdateReviewState()
		if first && len(current.Candidates) > 0 {
			// Only the first publication is news. The owner's own answers get no push.
			if e = tx.QueueCaptureNotifications(current.ID, current.Revision); e != nil {
				return e
			}
		}
		if e = tx.PutCapture(current); e != nil {
			return e
		}
		if _, e = tx.PutEntry(domain.Entry{ID: current.EntryID, Kind: "capture", Text: "Review the email details.", Capture: &current}); e != nil {
			return e
		}
		if review.HasOpenQuestions(current) {
			return review.PostQuestions(tx, current)
		}
		return nil
	})
	if e != nil {
		return failed(e)
	}
	return nil
}
// fail ends an attempt and never schedules another: the owner retries from the app.
func (s *Service) fail(c domain.Capture, cause error) error {
	err := s.repository.Transact(func(tx port.Transaction) error {
		current, e := tx.Capture(c.ID)
		if e != nil {
			return e
		}
		if current.Deleted || current.Revision != c.Revision || current.LeaseUntil != c.LeaseUntil {
			return domain.ErrConflict
		}
		current.Attempts++
		current.LeaseUntil, current.NextAt = 0, 0
		current.Status = "failed"
		current.Error = failureMessage(cause)
		if e = tx.PutCapture(current); e != nil {
			return e
		}
		if e = tx.QueueCaptureNotifications(current.ID, current.Revision); e != nil {
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

// failureMessage is the fixed sentence the owner sees for a failed attempt.
func failureMessage(cause error) string {
	var invalid *domain.Invalid
	switch {
	case errors.As(cause, &invalid):
		return invalid.Message
	case errors.Is(cause, ErrDocument):
		return domain.ReasonUnparsed.Sentence()
	}
	return domain.ReasonOf(cause).Sentence()
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
		body, e := s.pushBody(delivery.CaptureID)
		if e != nil {
			cancel()
			return e
		}
		invalid, sendErr := s.Sender.Send(ctx, delivery.Token, map[string]string{"capture_id": delivery.CaptureID, "type": "capture_review", "title": "Minder", "body": body})
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

func (s *Service) releaseClaim(claim domain.Capture) error {
	return s.repository.Transact(func(tx port.Transaction) error {
		current, e := tx.Capture(claim.ID)
		if e != nil {
			return e
		}
		if current.Deleted || current.Revision != claim.Revision || current.LeaseUntil != claim.LeaseUntil {
			return nil
		}
		current.Status = "received"
		current.LeaseUntil = 0
		current.NextAt = 0
		return tx.PutCapture(current)
	})
}

// supersededSlot returns the position of a superseded candidate with this id, or -1.
func supersededSlot(candidates []domain.CaptureCandidate, id string) int {
	for index, candidate := range candidates {
		if candidate.ID == id && candidate.State == "superseded" {
			return index
		}
	}
	return -1
}

// pushBody words the push for the capture's current state.
func (s *Service) pushBody(id string) (string, error) {
	c, e := s.repository.Capture(id)
	if e != nil {
		return "", e
	}
	return review.PushBody(c), nil
}
