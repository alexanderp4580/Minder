package capture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"net/mail"
	"time"
)

func (s *Service) Ingest(ctx context.Context, envelope domain.CaptureEnvelope, raw []byte) (id string, duplicate bool, err error) {
	if len(raw) > domain.MaxEmailBytes {
		return "", false, &domain.Invalid{Field: "email", Message: "The email exceeds 20 MiB."}
	}
	sum := sha256.Sum256(raw)
	id = hex.EncodeToString(sum[:])
	err = s.repository.Transact(func(tx port.Transaction) error {
		if r, e := tx.Receipt(envelope.RequestID); e == nil {
			if r.Digest != id {
				return domain.ErrConflict
			}
			c, e := tx.Capture(r.CaptureID)
			if e != nil {
				return e
			}
			if c.Deleted {
				return domain.ErrNotFound
			}
			duplicate = true
			return nil
		} else if !errors.Is(e, domain.ErrNotFound) {
			return e
		}
		if c, e := tx.Capture(id); e == nil {
			if c.Deleted {
				return domain.ErrNotFound
			}
			duplicate = true
			return tx.PutReceipt(domain.CaptureReceipt{ID: envelope.RequestID, Digest: id, CaptureID: id})
		} else if !errors.Is(e, domain.ErrNotFound) {
			return e
		}
		_, n, e := s.files.SaveOriginal(ctx, id, bytes.NewReader(raw))
		if e != nil {
			return e
		}
		subject := ""
		if msg, e := mail.ReadMessage(bytes.NewReader(raw)); e == nil {
			subject = msg.Header.Get("Subject")
		}
		c := domain.Capture{ID: id, Status: "received", Revision: 1, Sender: envelope.From, Subject: subject, ReceivedAt: time.Now().UTC().Format(time.RFC3339), Attachments: []domain.CaptureAttachment{{ID: "original", Name: "original.eml", MediaType: "message/rfc822", Size: n, SHA256: id}}, Candidates: []domain.CaptureCandidate{}, Coverage: []domain.CaptureCoverage{}}
		c.EntryID = domain.NewID()
		if _, e = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Email received. Processing.", Capture: &c}); e != nil {
			return e
		}
		if e = tx.PutCapture(c); e != nil {
			return e
		}
		return tx.PutReceipt(domain.CaptureReceipt{ID: envelope.RequestID, Digest: id, CaptureID: id})
	})
	return
}
