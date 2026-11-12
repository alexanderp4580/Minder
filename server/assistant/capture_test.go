package assistant_test

import (
	"context"
	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"testing"
)

type captureModel struct{ messages []assistant.Message }

func (m *captureModel) Complete(_ context.Context, messages []assistant.Message) (assistant.Response, error) {
	m.messages = messages
	return assistant.Response{Calls: []assistant.Call{{Name: "extract_capture", Arguments: `{"candidates":[{"proposal":{"action":"create_item","title":"Appointment","kind":"event"},"question":"What date is the appointment?","evidence":[{"part":"1","excerpt":"appointment"}]}],"questions":[]}`}}, CostUSD: 0.001}, nil
}
func TestCaptureExtractionKeepsQuestionsAndDoesNotSave(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	m := &captureModel{}
	svc := assistant.New(db, command.New(db), m, 5, 1)
	result, e := svc.Extract(context.Background(), domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1", Text: "appointment"}}}, nil)
	if e != nil || len(result.Candidates) != 1 || result.Candidates[0].Question == "" {
		t.Fatal(result, e)
	}
	items, _ := db.Items()
	if len(items) != 0 {
		t.Fatal("model saved items")
	}
	cost, e := db.MonthCost()
	if e != nil || cost != 0.001 {
		t.Fatal(cost, e)
	}
}

type duplicateCaptureModel struct{}

func (duplicateCaptureModel) Complete(context.Context, []assistant.Message) (assistant.Response, error) {
	return assistant.Response{Calls: []assistant.Call{{Name: "extract_capture", Arguments: `{"candidates":[{"id":"hotel","proposal":{"action":"create_item","title":"Hotel","kind":"event","start_date":"2026-11-12","end_date":"2026-11-15"},"evidence":[{"part":"1","excerpt":"Hotel"}]}],"questions":[]}`}}, CostUSD: 0.001}, nil
}
func TestExistingBookingRequiresDuplicateReview(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	commands := command.New(db)
	preview, e := commands.Preview(domain.Proposal{Action: "create_item", Title: "Hotel", Kind: "event", StartDate: "2026-11-12", EndDate: "2026-11-15"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = commands.Confirm(preview.Change.ID); e != nil {
		t.Fatal(e)
	}
	svc := assistant.New(db, commands, duplicateCaptureModel{}, 5, 1)
	out, e := svc.Extract(context.Background(), domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1", Text: "Hotel"}}}, nil)
	if e != nil || out.Candidates[0].Question == "" {
		t.Fatal("duplicate was not flagged", out, e)
	}
}

func TestPendingEmailBookingRequiresDuplicateReview(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	e = db.Transact(func(tx port.Transaction) error {
		return tx.PutCapture(domain.Capture{ID: "earlier", Status: "ready-for-review", Revision: 1, Candidates: []domain.CaptureCandidate{{ID: "hotel", State: "pending", Proposal: domain.Proposal{Title: "Hotel", Kind: "event", StartDate: "2026-11-12", EndDate: "2026-11-15"}}}})
	})
	if e != nil {
		t.Fatal(e)
	}
	svc := assistant.New(db, command.New(db), duplicateCaptureModel{}, 5, 1)
	out, e := svc.Extract(context.Background(), domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1", Text: "Hotel"}}}, nil)
	if e != nil || out.Candidates[0].Question == "" {
		t.Fatal("pending duplicate was not flagged", out, e)
	}
}
