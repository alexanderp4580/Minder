package capture_test

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

var tinyGIF, _ = base64.StdEncoding.DecodeString("R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7")

const mixedMail = "From: clinic@example.com\r\nSubject: Dentist visit\r\nContent-Type: multipart/mixed; boundary=x\r\n\r\n" +
	"--x\r\nContent-Type: text/plain\r\n\r\nBring your insurance card.\r\n" +
	"--x\r\nContent-Type: text/calendar\r\n\r\nBEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:visit-1\r\nDTSTART:20261120T100000Z\r\nSUMMARY:Dentist\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n" +
	"--x\r\nContent-Type: image/gif\r\nContent-Disposition: attachment; filename=map.gif\r\nContent-Transfer-Encoding: base64\r\n\r\n" + "R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" + "\r\n--x--\r\n"

func TestReadingAnEmailReturnsItsBodyCalendarAndImageParts(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "m", mixedMail)
	f.prepare(t)
	doc, err := f.svc.ReadSource(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var text, calendar, image bool
	for _, part := range doc.Document.Parts {
		text = text || strings.Contains(part.Text, "Bring your insurance card.")
		calendar = calendar || strings.Contains(part.Text, "UID:visit-1")
		image = image || (part.MediaType == "image/gif" && string(part.Data) == string(tinyGIF))
	}
	if doc.Source.ID != id || doc.Source.Kind != domain.SourceEmail || doc.Document.Subject != "Dentist visit" || !text || !calendar || !image || len(doc.Document.Attachments) != 2 {
		t.Fatalf("%+v", doc)
	}
}

func TestReadingChatTextReturnsOneTextPartAndNoEmailStructure(t *testing.T) {
	f := newFixture(t)
	entry, _ := f.db.AddEntry(domain.Entry{Kind: "user", Text: "Dentist on Friday at 10\nBring the card"})
	doc, err := f.svc.ReadSource(context.Background(), entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	parts := doc.Document.Parts
	if doc.Source.Kind != domain.SourceChat || doc.Source.EntryID != entry.ID || len(parts) != 1 || parts[0].Text != entry.Text || parts[0].ID != "text" || len(doc.Document.Attachments) != 0 || doc.Document.Subject != "Dentist on Friday at 10" {
		t.Fatalf("%+v", doc)
	}
}

func TestOnlyUserTextAndKnownSourcesCanBeRead(t *testing.T) {
	f := newFixture(t)
	reply, _ := f.db.AddEntry(domain.Entry{Kind: "minder", Text: "Saved."})
	for _, id := range []string{"missing", reply.ID} {
		if _, err := f.svc.ReadSource(context.Background(), id); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("%s: %v", id, err)
		}
	}
}

func TestADeletedSourceIsUnavailableEvenToAStaleReference(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "m", mixedMail)
	f.prepare(t)
	if _, err := f.svc.ReadSource(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Transact(func(tx port.Transaction) error { // what a confirmed deletion records
		ref, err := tx.Source(id)
		if err != nil {
			return err
		}
		ref.Deleted = true
		return tx.PutSource(ref)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ReadSource(context.Background(), id); !errors.Is(err, domain.ErrSourceUnavailable) {
		t.Fatal(err)
	}
	if found, _ := f.svc.FindSources(context.Background(), "dentist"); len(found) != 0 {
		t.Fatalf("a deleted source was found: %+v", found)
	}
}

func TestADeletedCaptureIsUnavailableEvenWithoutAFlaggedReference(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "m", mixedMail)
	f.prepare(t)
	f.db.Transact(func(tx port.Transaction) error {
		c, _ := tx.Capture(id)
		c.Deleted = true
		return tx.PutCapture(c)
	})
	if _, err := f.svc.ReadSource(context.Background(), id); !errors.Is(err, domain.ErrSourceUnavailable) {
		t.Fatal(err)
	}
}

func TestAMissingOriginalIsAnExplicitUnreadableError(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "m", mixedMail)
	f.prepare(t)
	if err := f.files.DeleteSource(id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ReadSource(context.Background(), id); !errors.Is(err, domain.ErrSourceUnreadable) {
		t.Fatal(err)
	}
}

func TestUnreadableAttachmentsAppearInTheCoverage(t *testing.T) {
	f := newFixture(t)
	raw := "From: a@example.com\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=ticket.bin\r\n\r\nBinary"
	id := f.ingest(t, "m", raw)
	f.prepare(t)
	doc, err := f.svc.ReadSource(context.Background(), id)
	if err != nil || len(doc.Document.Coverage) != 1 || doc.Document.Coverage[0].Status == "parsed" {
		t.Fatalf("%+v %v", doc.Document.Coverage, err)
	}
}

func TestReadingChangesNothing(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "m", mixedMail)
	f.prepare(t)
	entry, _ := f.db.AddEntry(domain.Entry{Kind: "user", Text: "Dentist on Friday"})
	before, _ := f.svc.Get(id)
	history, _ := f.db.History(0, 100)
	items, _ := f.db.Items()
	for _, source := range []string{id, entry.ID} {
		if _, err := f.svc.ReadSource(context.Background(), source); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.FindSources(context.Background(), "dentist"); err != nil {
		t.Fatal(err)
	}
	after, _ := f.svc.Get(id)
	historyAfter, _ := f.db.History(0, 100)
	itemsAfter, _ := f.db.Items()
	if !reflect.DeepEqual(before, after) || len(history) != len(historyAfter) || len(items) != len(itemsAfter) {
		t.Fatal("a read changed the capture, the chat or the items")
	}
	deliver(f.svc)
	if len(f.push.bodies) != 0 {
		t.Fatalf("a read queued a push: %q", f.push.bodies)
	}
	if turn, err := f.db.ClaimAgentTurn(time.Now()); err != nil || turn.Kind != domain.TurnSourceArrival {
		t.Fatalf("a read queued a turn: %+v %v", turn, err)
	}
}
