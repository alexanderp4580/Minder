package store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
)

func TestTransactRollsBackItemsEventsAndAudit(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	stopped := errors.New("stop transaction")
	err = database.Transact(func(tx port.Transaction) error {
		if err := tx.PutItem(domain.Item{ID: "item", Title: "Task", Status: "open", Version: 1}); err != nil {
			return err
		}
		if _, err := tx.PutEntry(domain.Entry{ID: "entry", Kind: "result", Text: "Saved"}); err != nil {
			return err
		}
		if err := tx.PutAudit(domain.Audit{ID: "audit", Actor: "owner"}); err != nil {
			return err
		}
		return stopped
	})
	if !errors.Is(err, stopped) {
		t.Fatalf("want rollback cause, got %v", err)
	}
	items, err := database.Items()
	if err != nil || len(items) != 0 {
		t.Fatalf("items survived: %v %v", items, err)
	}
	history, err := database.History(0, 20)
	if err != nil || len(history) != 0 {
		t.Fatalf("events survived: %v %v", history, err)
	}
	count, err := database.AuditCount()
	if err != nil || count != 0 {
		t.Fatalf("audit survived: %v %v", count, err)
	}
}
