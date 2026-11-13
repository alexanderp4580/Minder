// Package port defines application-owned persistence contracts.
package port

import (
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

// Purged lists what PurgeItem removed that the caller must act on.
type Purged struct {
	// EntryIDs are the chat entries removed with the item.
	EntryIDs []string
	// OrphanSources are email sources that no remaining item uses.
	OrphanSources []string
}

type Items interface {
	Item(string) (domain.Item, error)
	PutItem(domain.Item) error
	DeleteItem(string) error
	// PurgeItem deletes an item with its reminders, earlier changes, audit rows,
	// chat entries and source links. The pending change keepPending and its entries stay.
	PurgeItem(id, keepPending string) (Purged, error)
}
type Profiles interface {
	Profile(string) (domain.Profile, error)
	PutProfile(domain.Profile) error
	DeleteProfile(string) error
	ListNames() ([]string, error)
}
type Changes interface {
	Pending(string) (domain.Pending, error)
	PutPending(domain.Pending) error
	Audit(string) (domain.Audit, error)
	PutAudit(domain.Audit) error
}
type Entries interface {
	Entry(string) (domain.Entry, error)
	PutEntry(domain.Entry) (domain.Entry, error)
}

type Reminders interface {
	DueSchedules(time.Time) ([]domain.ReminderSchedule, error)
	Schedule(string) (domain.ReminderSchedule, error)
	UpdateSchedule(domain.ReminderSchedule) error
	Occurrence(string) (domain.Occurrence, error)
	PutOccurrence(domain.Occurrence) error
	CloseReminders(string, string) error
	QueueDeliveries(string, time.Time) error
}

// Transaction is a single atomic write boundary. An error rolls back every change.
type Transaction interface {
	Captures
	Items
	Profiles
	Changes
	Entries
	Reminders
}
type UnitOfWork interface {
	Transact(func(Transaction) error) error
}
