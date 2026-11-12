package command_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
)

type unavailableItemRepository struct {
	*store.Store
	failure error
}
type unavailableItemTransaction struct {
	port.Transaction
	failure error
}

func (tx unavailableItemTransaction) Item(string) (domain.Item, error) {
	return domain.Item{}, tx.failure
}
func (r unavailableItemRepository) Transact(fn func(port.Transaction) error) error {
	return r.Store.Transact(func(tx port.Transaction) error { return fn(unavailableItemTransaction{tx, r.failure}) })
}
func TestConfirmPropagatesStorageFailureInsteadOfInventingAConflict(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	service := command.New(database)
	preview, err := service.Preview(domain.Proposal{Action: "create_item", Title: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Confirm(preview.Change.ID); err != nil {
		t.Fatal(err)
	}
	items, err := database.Items()
	if err != nil {
		t.Fatal(err)
	}
	preview, err = service.Preview(domain.Proposal{Action: "update_item", ItemID: items[0].ID, ExpectedVersion: 1, Title: "Changed"})
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("storage unavailable")
	if _, err = command.New(unavailableItemRepository{database, failure}).Confirm(preview.Change.ID); !errors.Is(err, failure) {
		t.Fatalf("want storage error, got %v", err)
	}
	if _, err = service.Confirm(preview.Change.ID); err != nil {
		t.Fatalf("failed confirm changed pending state: %v", err)
	}
}
