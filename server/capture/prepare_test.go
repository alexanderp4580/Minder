package capture_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/capture"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/mailparse"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
)

func TestPreparingASourceStoresItsDetailsAndQueuesOneArrivalTurn(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "r1", plainMail)
	f.prepare(t)
	c, _ := f.svc.Get(id)
	if !c.Prepared || c.Subject != "Checkups" || c.Status != "processing" || len(c.Candidates) != 0 {
		t.Fatalf("%+v", c)
	}
	turn, err := f.db.ClaimAgentTurn(time.Now())
	if err != nil || turn.ID != domain.ArrivalTurn(id, 1).ID || turn.Kind != domain.TurnSourceArrival || turn.SourceID != id {
		t.Fatalf("%+v %v", turn, err)
	}
	found, _ := f.db.FindSources("checkups")
	if len(found) != 1 || found[0].ID != id || found[0].Label != "Checkups" {
		t.Fatalf("%+v", found)
	}
	if err = f.svc.ProcessNext(context.Background(), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("a prepared source was claimed again: %v", err)
	}
	if _, err = f.db.ClaimAgentTurn(time.Now().Add(time.Hour)); err != nil && !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestPreparationDecidesNothingAboutTheContent(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "r1", plainMail)
	f.prepare(t)
	items, _ := f.db.Items()
	c, _ := f.svc.Get(id)
	if len(items) != 0 || len(c.Candidates) != 0 || len(c.Questions) != 0 {
		t.Fatalf("preparation interpreted the email: %+v", c)
	}
}

func TestAnUnparsableOriginalFailsOnceWithoutQueueingATurn(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "bad", "no headers here")
	err := f.svc.ProcessNext(context.Background(), time.Now())
	if err == nil {
		t.Fatal("the worker must see the failure")
	}
	for _, later := range []time.Duration{time.Second, time.Hour} {
		if e := f.svc.ProcessNext(context.Background(), time.Now().Add(later)); e != nil {
			t.Fatalf("a failed email was claimed again: %v", e)
		}
	}
	c, _ := f.svc.Get(id)
	if c.Status != "failed" || c.Error != domain.ReasonUnparsed.Sentence() || c.Prepared {
		t.Fatalf("%+v", c)
	}
	if _, err = f.db.ClaimAgentTurn(time.Now()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("a turn was queued for an unreadable email: %v", err)
	}
	history, _ := f.db.History(0, 20)
	if last := history[len(history)-1]; last.Kind != "minder" || last.CaptureID != id || last.Text != domain.ReasonUnparsed.Sentence() {
		t.Fatalf("no visible failure report: %+v", last)
	}
	deliver(f.svc)
	if len(f.push.bodies) != 1 || f.push.bodies[0] != "Minder could not read an email." {
		t.Fatalf("%q", f.push.bodies)
	}
}

func TestRetryPreparesAgainAndQueuesTheNewRevision(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "r1", plainMail)
	f.prepare(t)
	f.db.Transact(func(tx port.Transaction) error {
		c, _ := tx.Capture(id)
		c.Status, c.Error = "failed", "x"
		return tx.PutCapture(c)
	})
	if _, err := f.svc.Retry(id); err != nil {
		t.Fatal(err)
	}
	f.prepare(t)
	f.db.Transact(func(tx port.Transaction) error {
		first, _ := tx.AgentTurn(domain.ArrivalTurn(id, 1).ID)
		first.State = domain.TurnFailed
		return tx.PutAgentTurn(first)
	})
	turn, err := f.db.ClaimAgentTurn(time.Now())
	if err != nil || turn.ID != domain.ArrivalTurn(id, 2).ID {
		t.Fatalf("%+v %v", turn, err)
	}
}

func TestStartupReportsAnEarlierFailureOnceWithoutPushingAgain(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "r1", plainMail)
	if err := f.db.Transact(func(tx port.Transaction) error {
		c, err := tx.Capture(id)
		if err != nil {
			return err
		}
		c.Status, c.Error = "failed", domain.ReasonEvidence.Sentence()
		if err = tx.PutCapture(c); err != nil {
			return err
		}
		_, err = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: c.Error, Capture: &c})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	deliver(f.svc)
	history, _ := f.db.History(0, 100)
	first := history[len(history)-1]
	if first.Kind != "minder" || first.CaptureID != id || first.Text != domain.ReasonEvidence.Sentence() {
		t.Fatalf("missing recovery report: %+v", first)
	}
	deliver(f.svc)
	history, _ = f.db.History(0, 100)
	if last := history[len(history)-1]; last.ID != first.ID || last.Cursor != first.Cursor {
		t.Fatal("restart repeated or moved an existing failure report")
	}
	if len(f.push.bodies) != 0 {
		t.Fatalf("recovery pushed again: %q", f.push.bodies)
	}
}

// slowParser waits for cancellation, like a parse interrupted by shutdown.
type slowParser struct{ entered chan struct{} }

func (p slowParser) Parse(ctx context.Context, _ io.Reader) (domain.CaptureDocument, error) {
	close(p.entered)
	<-ctx.Done()
	return domain.CaptureDocument{}, ctx.Err()
}

func TestShutdownReleasesTheSourceWithoutConsumingRetry(t *testing.T) {
	f := newFixture(t)
	parser := slowParser{make(chan struct{})}
	f.svc.Parser = parser
	id := f.ingest(t, "cancel", plainMail)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- f.svc.ProcessNext(ctx, time.Now()) }()
	<-parser.entered
	cancel()
	<-done
	c, err := f.svc.Get(id)
	if err != nil || c.Status != "received" || c.Attempts != 0 || c.LeaseUntil != 0 || c.NextAt != 0 || c.Prepared {
		t.Fatalf("shutdown consumed retry: %+v %v", c, err)
	}
}

type blockedFiles struct {
	capture.Files
	entered, release chan struct{}
}

func (f *blockedFiles) Save(ctx context.Context, source, id string, raw io.Reader, limit int64) (string, int64, error) {
	close(f.entered)
	<-f.release
	return f.Files.Save(ctx, source, id, raw, limit)
}

func TestDeleteDuringAttachmentWriteDoesNotResurrectFiles(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files, _ := mailfiles.New(t.TempDir())
	blocked := &blockedFiles{Files: files, entered: make(chan struct{}), release: make(chan struct{})}
	svc := capture.New(db, blocked)
	svc.Parser = &mailparse.Parser{}
	raw := []byte("From: owner@example.com\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=ticket.bin\r\n\r\nAttachment")
	id, _, err := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "race-delete"}, raw)
	if err != nil {
		t.Fatal(err)
	}
	processed := make(chan struct{})
	go func() { defer close(processed); _ = svc.ProcessNext(context.Background(), time.Now()) }()
	<-blocked.entered
	confirmed := make(chan struct{})
	go func() {
		defer close(confirmed)
		_ = db.Transact(func(tx port.Transaction) error { // what a confirmed source deletion does
			c, _ := tx.Capture(id)
			if err := tx.PutCapture(domain.Capture{ID: id, Deleted: true, Revision: c.Revision + 1, Status: "reviewed", EntryID: c.EntryID}); err != nil {
				return err
			}
			return tx.RedactCapture(id)
		})
	}()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	select {
	case <-confirmed:
		svc.Run(canceled)
	case <-time.After(100 * time.Millisecond):
	}
	close(blocked.release)
	<-processed
	<-confirmed
	svc.Run(canceled)
	if f, err := files.Open(id, "1"); err == nil {
		f.Close()
		t.Fatal("attachment was recreated after deletion")
	}
}
