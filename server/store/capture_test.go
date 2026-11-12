package store_test

import (
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"testing"
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
