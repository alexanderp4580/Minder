package command

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

func setColorChoice(change *domain.Change, item domain.Item) {
	if change.Action == "create_item" || change.Action == "update_item" {
		change.Color = item.Color
	}
}

// SetColor edits an item proposal, never the saved item. Confirm retains its version check.
func (c *Service) SetColor(id, color string) (out domain.Entry, err error) {
	if !domain.ValidColor(color) {
		return out, &domain.Invalid{Field: "color", Message: "Choose Blue, Teal, Green, Amber, Lilac or Slate."}
	}
	err = c.repository.Transact(func(t port.Transaction) error {
		p, e := t.Pending(id)
		if e != nil {
			return e
		}
		if p.State != "pending" || (p.Proposal.Action != "create_item" && p.Proposal.Action != "update_item") {
			return domain.ErrConflict
		}
		if e := validateCaptureRevision(t, p.Proposal); e != nil {
			return e
		}
		entry, e := t.Entry(p.EntryID)
		if e != nil {
			return e
		}
		p.After.Color = color
		p.Proposal.Color = color
		old := domain.Item{}
		if p.Before != nil {
			old = *p.Before
		}
		entry.Change.Fields = itemFields(t, old, p.After)
		setColorChoice(entry.Change, p.After)
		if p.Proposal.CaptureID != "" {
			capture, e := t.Capture(p.Proposal.CaptureID)
			if e != nil {
				return e
			}
			if capture.Deleted {
				return domain.ErrConflict
			}
			for n := range capture.Candidates {
				if capture.Candidates[n].ID == p.Proposal.CandidateID {
					capture.Candidates[n].Proposal.Color = color
				}
			}
			if e = t.PutCapture(capture); e != nil {
				return e
			}
		}
		if out, e = t.PutEntry(entry); e != nil {
			return e
		}
		return t.PutPending(p)
	})
	return
}
