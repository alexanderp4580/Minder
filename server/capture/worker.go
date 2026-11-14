package capture

import (
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

	if err := s.restoreFailureReports(); err != nil {
		slog.Warn("email failure report recovery failed", "error_code", domain.ErrorCode(err))
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

// ProcessNext prepares the oldest received source: it parses the original, stores the
// attachments and queues one agent turn to review it. It decides nothing about the content.
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
	if s.Parser == nil || s.files == nil {
		return failed(errDocument)
	}
	f, e := s.files.Open(c.ID, "original")
	if e != nil {
		return failed(e)
	}
	document, e := s.Parser.Parse(ctx, f)
	f.Close()
	if e != nil {
		return failed(errDocument)
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
		current.Prepared = true
		current.LeaseUntil, current.NextAt = 0, 0
		current.Error = ""
		if e = tx.PutCapture(current); e != nil {
			return e
		}
		ref, e := tx.Source(c.ID)
		if e != nil {
			return e
		}
		ref.Label = document.Subject
		if e = tx.PutSource(ref); e != nil {
			return e
		}
		if _, e = tx.PutEntry(domain.Entry{ID: current.EntryID, Kind: "capture", Text: "Reading the email.", Capture: &current}); e != nil {
			return e
		}
		return tx.PutAgentTurn(domain.ArrivalTurn(current.ID, current.Revision))
	})
	if e != nil {
		return failed(e)
	}
	slog.Info("email prepared", "source_id", c.ID, "revision", c.Revision, "parts", len(document.Parts), "attachments", len(document.Attachments))
	if s.Diagnostics != nil {
		s.Diagnostics(domain.WithTrace(ctx, domain.ArrivalTurn(c.ID, c.Revision).ID), "email_prepared", map[string]any{"source_id": c.ID, "revision": c.Revision, "document": document})
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
		return review.Fail(tx, current, cause)
	})
	if err != nil {
		return err
	}
	slog.Warn("email attempt failed", "source_id", c.ID, "revision", c.Revision, "reason", string(domain.ReasonOf(cause)), "error_code", domain.ErrorCode(cause))
	return cause
}

// errDocument is the failure for an original that cannot be parsed or has nowhere to be read.
var errDocument = &domain.ModelError{Message: "The email could not be parsed. The original is available.", Reason: domain.ReasonUnparsed}

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

// pushBody words the push for the capture's current state.
func (s *Service) pushBody(id string) (string, error) {
	c, e := s.repository.Capture(id)
	if e != nil {
		return "", e
	}
	return review.PushBody(c), nil
}
