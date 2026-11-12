package store_test

import (
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"testing"
	"time"
)

func TestCaptureTransactionRollback(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	sentinel := errors.New("rollback")
	e = s.Transact(func(tx port.Transaction) error {
		if e := tx.PutCapture(domain.Capture{ID: "source", Status: "received", Revision: 1}); e != nil {
			return e
		}
		return sentinel
	})
	if !errors.Is(e, sentinel) {
		t.Fatal(e)
	}
	e = s.Transact(func(tx port.Transaction) error { _, e := tx.Capture("source"); return e })
	if !errors.Is(e, domain.ErrNotFound) {
		t.Fatal("rollback lost", e)
	}
}
func TestCaptureReceiptAndLinks(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	e = s.Transact(func(tx port.Transaction) error {
		if e := tx.PutCapture(domain.Capture{ID: "source", Status: "received", Revision: 1}); e != nil {
			return e
		}
		if e := tx.PutReceipt(domain.CaptureReceipt{ID: "request", Digest: "digest", CaptureID: "source"}); e != nil {
			return e
		}
		if e := tx.LinkSource("item", "source"); e != nil {
			return e
		}
		return tx.LinkSource("item", "source")
	})
	if e != nil {
		t.Fatal(e)
	}
	e = s.Transact(func(tx port.Transaction) error {
		r, e := tx.Receipt("request")
		if e != nil || r.CaptureID != "source" {
			t.Fatal(r, e)
		}
		ids, e := tx.SourcesForItem("item")
		if e != nil || len(ids) != 1 {
			t.Fatal(ids, e)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}

func TestClaimLeaseRecovery(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	now := time.Now()
	if e = db.Transact(func(tx port.Transaction) error {
		return tx.PutCapture(domain.Capture{ID: "lease", Status: "received", Revision: 1})
	}); e != nil {
		t.Fatal(e)
	}
	first, e := db.ClaimCapture(now)
	if e != nil || first.LeaseUntil <= now.Unix() {
		t.Fatal(first, e)
	}
	if _, e = db.ClaimCapture(now); !errors.Is(e, domain.ErrNotFound) {
		t.Fatal("double claim", e)
	}
	recovered, e := db.ClaimCapture(now.Add(6 * time.Minute))
	if e != nil || recovered.ID != first.ID {
		t.Fatal(recovered, e)
	}
}

func TestReviewedCaptureDoesNotSendStaleNotification(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = reminder.New(db).RegisterDevice("device", "fixture-token"); e != nil {
		t.Fatal(e)
	}
	e = db.Transact(func(tx port.Transaction) error {
		if e := tx.PutCapture(domain.Capture{ID: "reviewed", Revision: 1, Status: "reviewed"}); e != nil {
			return e
		}
		return tx.QueueCaptureNotifications("reviewed", 1)
	})
	if e != nil {
		t.Fatal(e)
	}
	deliveries, e := db.CaptureDeliveries(time.Now())
	if e != nil || len(deliveries) != 0 {
		t.Fatal("stale review notification", deliveries, e)
	}
}
