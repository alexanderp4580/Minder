package review

import (
	"github.com/alexanderp4580/Minder/server/port"
)

// WithdrawPreview fails a still-pending preview so it can no longer be confirmed. A missing
// change, or one already decided, is left alone.
func WithdrawPreview(tx port.Transaction, changeID string) error {
	if changeID == "" {
		return nil
	}
	p, err := tx.Pending(changeID)
	if err != nil || p.State != "pending" {
		return err
	}
	p.State = "failed"
	if err = tx.PutPending(p); err != nil {
		return err
	}
	entry, err := tx.Entry(p.EntryID)
	if err != nil {
		return err
	}
	entry.Change.State = "failed"
	_, err = tx.PutEntry(entry)
	return err
}
