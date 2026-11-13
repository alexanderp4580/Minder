package capture_test

import (
	"context"
	"github.com/alexanderp4580/Minder/server/capture"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/mailparse"
	"github.com/alexanderp4580/Minder/server/store"
	"io"
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

func TestTargetedAnswerSupersedesChosenPreview(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, questionsExtractor{})
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, _ := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "edit"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	c, _ := svc.Get(id)
	old := c.Candidates[0].ChangeID
	if _, e = svc.AnswerCandidate(context.Background(), id, c.Revision, "ready", "Change the deadline"); e != nil {
		t.Fatal(e)
	}
	if _, e = command.New(db).Confirm(old); e == nil {
		t.Fatal("stale preview accepted")
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
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := mailfiles.New(t.TempDir())
	blocked := &blockedFiles{Files: files, entered: make(chan struct{}), release: make(chan struct{})}
	svc := capture.New(db, blocked, extracted{})
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	raw := []byte("From: owner@example.com\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=ticket.bin\r\n\r\nAttachment")
	id, _, e := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "race-delete"}, raw)
	if e != nil {
		t.Fatal(e)
	}
	c, _ := svc.Get(id)
	commands := command.New(db)
	preview, e := commands.Preview(domain.Proposal{Action: "delete_capture", CaptureID: id, ExpectedVersion: c.Revision})
	if e != nil {
		t.Fatal(e)
	}
	processed := make(chan struct{})
	go func() { defer close(processed); _ = svc.ProcessNext(context.Background(), time.Now()) }()
	<-blocked.entered
	confirmed := make(chan struct{})
	go func() { defer close(confirmed); _, _ = commands.Confirm(preview.Change.ID) }()
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
	if f, e := files.Open(id, "1"); e == nil {
		f.Close()
		t.Fatal("attachment was recreated after deletion")
	}
}

type cancelExtraction struct{ entered chan struct{} }

func (e cancelExtraction) Extract(ctx context.Context, _ domain.CaptureDocument, _ []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	close(e.entered)
	<-ctx.Done()
	return domain.CaptureExtraction{}, ctx.Err()
}
func TestShutdownReleasesCaptureWithoutConsumingRetry(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := mailfiles.New(t.TempDir())
	extractor := cancelExtraction{make(chan struct{})}
	svc := capture.New(db, files, extractor)
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, e := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "cancel"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- svc.ProcessNext(ctx, time.Now()) }()
	<-extractor.entered
	cancel()
	<-done
	c, e := svc.Get(id)
	if e != nil || c.Status != "received" || c.Attempts != 0 || c.LeaseUntil != 0 || c.NextAt != 0 {
		t.Fatalf("shutdown consumed retry: %+v %v", c, e)
	}
}

type omittingExtractor struct{}

func (omittingExtractor) Extract(ctx context.Context, doc domain.CaptureDocument, answers []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	if len(answers) > 0 {
		return domain.CaptureExtraction{}, nil
	}
	return (questionsExtractor{}).Extract(ctx, doc, answers)
}
func TestFollowupCannotSilentlyOmitQuestionedItem(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, omittingExtractor{})
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, _ := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "omit"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	c, _ := svc.Get(id)
	if _, e = svc.Answer(context.Background(), id, c.Revision, "21 November"); e != nil {
		t.Fatal(e)
	}
	if e = svc.ProcessNext(context.Background(), time.Now()); e == nil {
		t.Fatal("questioned item disappeared without a failure")
	}
}

func TestConfirmedItemExposesItsEmailSources(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, extracted{})
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, _ := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "source-link"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	c, _ := svc.Get(id)
	if _, e = command.New(db).Confirm(c.Candidates[0].ChangeID); e != nil {
		t.Fatal(e)
	}
	items, e := db.Items()
	if e != nil || len(items) != 1 || len(items[0].SourceIDs) != 1 || items[0].SourceIDs[0] != id {
		t.Fatal("item has no retrievable source", items, e)
	}
}

type emptyExtractor struct{}

func (emptyExtractor) Extract(context.Context, domain.CaptureDocument, []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	return domain.CaptureExtraction{}, nil
}

func TestDismissClosesAnOpenCaptureWithoutChangingItems(t *testing.T) {
	for name, extractor := range map[string]capture.Extractor{"candidates and questions": questionsExtractor{}, "zero items": emptyExtractor{}} {
		t.Run(name, func(t *testing.T) {
			db, e := store.Open(filepath.Join(t.TempDir(), "db"))
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			files, _ := mailfiles.New(t.TempDir())
			svc := capture.New(db, files, extractor)
			svc.Parser = &mailparse.Parser{}
			svc.Preview = command.PreviewIn
			id, _, _ := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "dismiss"}, []byte("From: owner@example.com\r\n\r\nHotel"))
			if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
				t.Fatal(e)
			}
			c, _ := svc.Get(id)
			if _, e = svc.Dismiss(context.Background(), id, c.Revision+1); e == nil {
				t.Fatal("stale revision dismissed")
			}
			entry, e := svc.Dismiss(context.Background(), id, c.Revision)
			if e != nil || entry.Capture == nil || entry.Capture.Status != "reviewed" {
				t.Fatal(entry, e)
			}
			c, _ = svc.Get(id)
			for _, candidate := range c.Candidates {
				if candidate.State != "discarded" {
					t.Fatal("candidate left open", candidate)
				}
				if candidate.ChangeID != "" {
					if _, e = command.New(db).Confirm(candidate.ChangeID); e == nil {
						t.Fatal("dismissed preview still confirmable")
					}
				}
			}
			if items, _ := db.Items(); len(items) != 0 {
				t.Fatal("dismiss changed items", items)
			}
			if f, _, e := svc.Attachment(id, "original"); e != nil {
				t.Fatal("source not kept", e)
			} else {
				f.Close()
			}
			if _, e = svc.Dismiss(context.Background(), id, c.Revision); e == nil {
				t.Fatal("reviewed capture dismissed twice")
			}
			if _, e = svc.Answer(context.Background(), id, c.Revision, "reopen"); e == nil {
				t.Fatal("answer reopened a dismissed capture")
			}
		})
	}
}
