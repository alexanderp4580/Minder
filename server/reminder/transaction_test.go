package reminder_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
)

type failingRepository struct {
	*store.Store
	failure error
}
type failingTransaction struct {
	port.Transaction
	failure error
}

func (t failingTransaction) PutEntry(entry domain.Entry) (domain.Entry, error) {
	if entry.Kind == "result" {
		return domain.Entry{}, t.failure
	}
	return t.Transaction.PutEntry(entry)
}
func (r failingRepository) Transact(fn func(port.Transaction) error) error {
	return r.Store.Transact(func(tx port.Transaction) error { return fn(failingTransaction{tx, r.failure}) })
}
func TestFailedReminderResultRollsBackCompletionAndCanRetry(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	err = database.Transact(func(tx port.Transaction) error {
		return tx.PutItem(domain.Item{ID: "item", Title: "Bins", Status: "open", Version: 1, RemindAt: now.Format(time.RFC3339), Timezone: "Europe/Amsterdam"})
	})
	if err != nil {
		t.Fatal(err)
	}
	service := reminder.New(database)
	if err = service.ScheduleDue(now); err != nil {
		t.Fatal(err)
	}
	active, err := service.Active()
	if err != nil || len(active) != 1 {
		t.Fatal(active, err)
	}
	id := active[0].Reminder.ID
	failure := errors.New("result write failed")
	if _, err = reminder.New(failingRepository{database, failure}).Act(id, "done", now); !errors.Is(err, failure) {
		t.Fatalf("want storage error, got %v", err)
	}
	items, err := database.Items()
	if err != nil || items[0].Status != "open" {
		t.Fatal("completion survived rollback", items, err)
	}
	active, err = service.Active()
	if err != nil || len(active) != 1 {
		t.Fatal("occurrence lost on rollback", err)
	}
	count, err := database.AuditCount()
	if err != nil || count != 0 {
		t.Fatal("audit survived rollback", count, err)
	}
	result, err := service.Act(id, "done", now)
	if err != nil || result.Change.UndoID == "" {
		t.Fatal(result, err)
	}
	again, err := service.Act(id, "done", now)
	if err != nil || again.ID != result.ID {
		t.Fatal("retry is not idempotent", again, err)
	}
	count, err = database.AuditCount()
	if err != nil || count != 1 {
		t.Fatal("retry duplicated audit", count, err)
	}
}
