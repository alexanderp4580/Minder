package capture_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/capture"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/mailparse"
	"github.com/alexanderp4580/Minder/server/store"
)

// pushes records the body of every push sent.
type pushes struct{ bodies []string }

func (p *pushes) Send(_ context.Context, _ string, data map[string]string) (bool, error) {
	p.bodies = append(p.bodies, data["body"])
	return false, nil
}

// fixture is a capture service over a real temporary database and file store.
type fixture struct {
	svc   *capture.Service
	db    *store.Store
	files *mailfiles.Store
	push  *pushes
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = db.RegisterDevice("phone", "token"); err != nil {
		t.Fatal(err)
	}
	files, err := mailfiles.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := capture.New(db, files)
	svc.Parser = &mailparse.Parser{}
	sender := &pushes{}
	svc.Sender = sender
	return fixture{svc, db, files, sender}
}

// ingest stores a synthetic email and returns its source ID.
func (f fixture) ingest(t *testing.T, request, raw string) string {
	t.Helper()
	id, _, err := f.svc.Ingest(context.Background(), domain.CaptureEnvelope{From: "owner@example.com", RequestID: request}, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// prepare runs the worker's preparation of the oldest received source.
func (f fixture) prepare(t *testing.T) {
	t.Helper()
	if err := f.svc.ProcessNext(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
}

// deliver runs one worker pass that sends the queued pushes and then stops.
func deliver(svc *capture.Service) {
	stopped, cancel := context.WithCancel(context.Background())
	cancel()
	svc.Run(stopped)
}

const plainMail = "Subject: Checkups\r\nFrom: owner@example.com\r\n\r\nHotel"
