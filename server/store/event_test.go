package store_test

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"testing"
	"time"
)

func TestCalendarStayOverlap(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	e = s.Write(func(tx *store.Tx) error {
		return tx.PutItem(domain.Item{ID: "hotel", Title: "Stay", Status: "open", StartDate: "2026-10-28", EndDate: "2026-11-03", Version: 1})
	})
	if e != nil {
		t.Fatal(e)
	}
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	items, e := s.ItemsBetween(from, from.AddDate(0, 0, 1))
	if e != nil || len(items) != 1 {
		t.Fatal(items, e)
	}
	items, e = s.ItemsBetween(from.AddDate(0, 0, 2), from.AddDate(0, 0, 3))
	if e != nil || len(items) != 0 {
		t.Fatal(items, e)
	}
}
