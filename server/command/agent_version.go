package command

import (
	"errors"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// useCurrentVersion sets the version an agent proposal expects to the item's stored version.
// The model names an item but does not have to echo its version. The preview still records the
// item as it was, and Confirm refuses the change if the item changed after that.
func useCurrentVersion(t port.Transaction, p *domain.Proposal) error {
	switch p.Action {
	case "update_item", "complete_item", "cancel_item", "delete_item":
	default:
		return nil
	}
	item, err := t.Item(p.ItemID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil // the preview reports the unknown item
	}
	if err != nil {
		return err
	}
	p.ExpectedVersion = item.Version
	return nil
}
