package capture_test

import (
	"context"
	"github.com/alexanderp4580/Minder/server/capture"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/mailparse"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"testing"
	"time"
)

type extracted struct{}

func (extracted) Extract(context.Context, domain.CaptureDocument, []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	return domain.CaptureExtraction{Candidates: []domain.CaptureCandidate{{Proposal: domain.Proposal{Action: "create_item", Title: "Hotel", Kind: "event", StartDate: "2026-11-12", EndDate: "2026-11-15"}, Evidence: []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}}}}, nil
}
func TestCaptureReviewBeforeSave(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, e := mailfiles.New(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	svc := capture.New(db, files, extracted{})
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, e := svc.Ingest(context.Background(), domain.CaptureEnvelope{From: "owner@example.com", RequestID: "review"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e != nil {
		t.Fatal(e)
	}
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	items, _ := db.Items()
	if len(items) != 0 {
		t.Fatal("extraction saved item")
	}
	c, e := svc.Get(id)
	if e != nil || len(c.Candidates) != 1 || c.Candidates[0].ChangeID == "" {
		t.Fatal(c, e)
	}
	commands := command.New(db)
	r, e := commands.Confirm(c.Candidates[0].ChangeID)
	if e != nil {
		t.Fatal(e)
	}
	again, e := commands.Confirm(c.Candidates[0].ChangeID)
	if e != nil || r.ID != again.ID {
		t.Fatal(again, e)
	}
	c, _ = svc.Get(id)
	if c.Candidates[0].State != "confirmed" {
		t.Fatal(c.Candidates)
	}
	if f, _, err := svc.Attachment(id, "original"); err != nil {
		t.Fatal(err)
	} else {
		f.Close()
	}
	if _, e = svc.Answer(context.Background(), id, c.Revision, "change it"); e == nil {
		t.Fatal("terminal candidate reopened")
	}
}

func TestSourceDeletionNeedsConfirmation(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, extracted{})
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, e := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "delete"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e != nil {
		t.Fatal(e)
	}
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	c, _ := svc.Get(id)
	commands := command.New(db)
	preview, e := commands.Preview(domain.Proposal{Action: "delete_capture", CaptureID: id, ExpectedVersion: c.Revision})
	if e != nil {
		t.Fatal(e)
	}
	f, _, e := svc.Attachment(id, "original")
	if e != nil {
		t.Fatal("deleted before confirm", e)
	}
	f.Close()
	if _, e = commands.Confirm(preview.Change.ID); e != nil {
		t.Fatal(e)
	}
	if _, _, e = svc.Attachment(id, "original"); e == nil {
		t.Fatal("deleted source accessible")
	}
	if _, e = commands.Confirm(c.Candidates[0].ChangeID); e == nil {
		t.Fatal("deleted source preview accepted")
	}
	history, _ := db.History(0, 200)
	for _, entry := range history {
		if entry.Capture != nil && (entry.Capture.Sender != "" || len(entry.Capture.Candidates) > 0) {
			t.Fatal("source data remained in history")
		}
	}
}

type questionsExtractor struct{}

func (questionsExtractor) Extract(_ context.Context, d domain.CaptureDocument, answers []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	ready := domain.CaptureCandidate{ID: "ready", Proposal: domain.Proposal{Action: "create_item", Title: "Pay invoice", DueDate: "2026-11-20"}, Evidence: []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}}
	missing := domain.CaptureCandidate{ID: "question", Proposal: domain.Proposal{Action: "create_item", Title: "Appointment", Kind: "event"}, Question: "What date?", Evidence: ready.Evidence}
	if len(answers) > 0 {
		missing.Question = ""
		missing.Proposal.StartDate = "2026-11-21"
	}
	return domain.CaptureExtraction{Candidates: []domain.CaptureCandidate{ready, missing}}, nil
}
func TestAnswerPreservesIndependentCandidate(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, questionsExtractor{})
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, _ := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "question"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	c, _ := svc.Get(id)
	readyID := c.Candidates[0].ChangeID
	if _, e = svc.Answer(context.Background(), id, c.Revision, "21 November"); e != nil {
		t.Fatal(e)
	}
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e = command.New(db).Confirm(readyID); e != nil {
		t.Fatal("independent preview invalidated", e)
	}
	c, _ = svc.Get(id)
	pending := 0
	for _, candidate := range c.Candidates {
		if candidate.State == "pending" {
			pending++
		}
	}
	if pending != 1 {
		t.Fatal("answer lost or duplicated candidates", c.Candidates)
	}
}
