package command

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// PreviewIn allows capture publication to share the command transaction.
func PreviewIn(tx port.Transaction, p domain.Proposal) (out domain.Entry, err error) {
	err = previewIn(tx, p, &out)
	return
}

func validateCaptureRevision(tx port.Transaction, p domain.Proposal) error {
	if p.CaptureID == "" {
		return nil
	}
	c, e := tx.Capture(p.CaptureID)
	if e != nil {
		return e
	}
	if c.Deleted {
		return domain.ErrConflict
	}
	for _, candidate := range c.Candidates {
		if candidate.ID == p.CandidateID && candidate.Revision == p.CaptureRevision && candidate.State == "pending" {
			return nil
		}
	}
	return domain.ErrConflict
}
func updateCaptureCandidate(tx port.Transaction, p domain.Pending, state string) error {
	if p.Proposal.CaptureID == "" {
		return nil
	}
	c, e := tx.Capture(p.Proposal.CaptureID)
	if e != nil {
		return e
	}
	for n := range c.Candidates {
		candidate := &c.Candidates[n]
		if candidate.ID == p.Proposal.CandidateID && candidate.Revision == p.Proposal.CaptureRevision {
			candidate.State = state
			if state == "confirmed" {
				candidate.ItemID = p.After.ID
				if e = tx.LinkSource(p.After.ID, c.ID); e != nil {
					return e
				}
			}
		}
	}
	c.UpdateReviewState()
	if e = tx.PutCapture(c); e != nil {
		return e
	}
	_, e = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: "Email review updated.", Capture: &c})
	return e
}

func previewCaptureDeletion(tx port.Transaction, p domain.Proposal) (domain.Entry, error) {
	c, e := tx.Capture(p.CaptureID)
	if e != nil {
		return domain.Entry{}, e
	}
	if c.Deleted || p.ExpectedVersion != c.Revision {
		return domain.Entry{}, domain.ErrConflict
	}
	id := domain.NewID()
	entry, e := tx.PutEntry(domain.Entry{Kind: "preview", Text: "Delete the original email and attachments. Confirmed items remain. Undo cannot restore the source; older backups keep their copies until expiry.", Change: &domain.Change{ID: id, Action: "delete_capture", ItemTitle: "Email source", State: "pending", Fields: []domain.Field{{Name: "source", Old: "Original email and attachments", New: "Deleted"}}}})
	if e != nil {
		return entry, e
	}
	return entry, tx.PutPending(domain.Pending{ID: id, Proposal: p, BeforeCapture: &c, State: "pending", EntryID: entry.ID})
}
func confirmCaptureDeletion(tx port.Transaction, p *domain.Pending, out *domain.Entry) error {
	c, e := tx.Capture(p.Proposal.CaptureID)
	if e != nil {
		return e
	}
	if c.Deleted || c.Revision != p.Proposal.ExpectedVersion {
		return domain.ErrConflict
	}
	tombstone := domain.Capture{ID: c.ID, Deleted: true, Revision: c.Revision + 1, Status: "reviewed", EntryID: c.EntryID}
	if e = tx.PutCapture(tombstone); e != nil {
		return e
	}
	if e = tx.RedactCapture(c.ID); e != nil {
		return e
	}
	audit := domain.Audit{ID: domain.NewID(), Actor: "owner", DeletedCaptureID: c.ID}
	if e = tx.PutAudit(audit); e != nil {
		return e
	}
	preview, e := tx.Entry(p.EntryID)
	if e != nil {
		return e
	}
	preview.Change.State = "confirmed"
	if _, e = tx.PutEntry(preview); e != nil {
		return e
	}
	*out, e = tx.PutEntry(domain.Entry{Kind: "result", Text: "Email source deleted. Confirmed items remain.", Change: &domain.Change{ID: p.ID, Action: "delete_capture", ItemTitle: "Email source", State: "confirmed"}})
	if e != nil {
		return e
	}
	p.State = "confirmed"
	p.ResultID = out.ID
	p.BeforeCapture = nil
	return tx.PutPending(*p)
}
