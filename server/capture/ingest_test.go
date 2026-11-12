package capture

import (
	"context"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"testing"
)

func TestIngestDurableDuplicate(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, e := mailfiles.New(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	s := New(db, files, nil)
	env := domain.CaptureEnvelope{From: "owner@example.com", To: "assistant@example.net", DMARC: "pass", RequestID: "first"}
	raw := []byte("From: owner@example.com\r\nSubject: Booking\r\n\r\nFlight")
	id, duplicate, e := s.Ingest(context.Background(), env, raw)
	if e != nil || duplicate {
		t.Fatal(id, duplicate, e)
	}
	env.RequestID = "retry"
	id2, duplicate, e := s.Ingest(context.Background(), env, raw)
	if e != nil || !duplicate || id2 != id {
		t.Fatal(id2, duplicate, e)
	}
	_, _, e = s.Ingest(context.Background(), env, append(raw, '!'))
	if e == nil {
		t.Fatal("replay conflict accepted")
	}
	c, e := db.Capture(id)
	if e != nil || c.Status != "received" {
		t.Fatal(c, e)
	}
	f, e := files.Open(id, "original")
	if e != nil {
		t.Fatal(e)
	}
	f.Close()
}
