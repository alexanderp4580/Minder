package review

import (
	"errors"
	"fmt"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// FailureMessage is the fixed sentence the owner sees for a failed source. Invalid input
// keeps its own message; every other cause is classified, so no model text or detail leaks.
func FailureMessage(cause error) string {
	var invalid *domain.Invalid
	if errors.As(cause, &invalid) {
		return invalid.Message
	}
	return domain.ReasonOf(cause).Sentence()
}

// Fail ends a source's attempt as failed: the card shows the sentence and keeps its Retry
// action, one chat message reports it at the reading position, and one push is queued.
// Nothing retries automatically. The caller checks the revision and stores nothing else.
func Fail(tx port.Transaction, c domain.Capture, cause error) error {
	c.Attempts++
	c.LeaseUntil, c.NextAt = 0, 0
	c.Status = "failed"
	c.Error = FailureMessage(cause)
	if err := tx.PutCapture(c); err != nil {
		return err
	}
	if err := tx.QueueCaptureNotifications(c.ID, c.Revision); err != nil {
		return err
	}
	if _, err := tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: c.Error, Capture: &c}); err != nil {
		return err
	}
	_, err := PostFailure(tx, c)
	return err
}

// PostFailure appends one visible report per failed revision. It reports whether it added one.
func PostFailure(tx port.Transaction, c domain.Capture) (bool, error) {
	id := fmt.Sprintf("capture-failure-%s-%d", c.ID, c.Revision)
	if _, err := tx.Entry(id); err == nil {
		return false, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return false, err
	}
	text := c.Error
	if text == "" {
		text = domain.ReasonOther.Sentence()
	}
	_, err := tx.PutEntry(domain.Entry{ID: id, Kind: "minder", CaptureID: c.ID, Text: text})
	return err == nil, err
}
