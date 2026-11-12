package assistant_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
	"strings"
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

type repairEvidenceModel struct {
	calls       int
	keepInvalid bool
}

func (m *repairEvidenceModel) Complete(_ context.Context, messages []assistant.Message) (assistant.Response, error) {
	m.calls++
	arguments := `{"candidates":[{"proposal":{"action":"create_item","title":"Setup appointment","kind":"event"},"question":"Which date?","evidence":[{"part":"1_2 (setup.ics)","excerpt":"SUMMARY:Setup appointment / END:VEVENT"}]}],"questions":[]}`
	if m.calls == 2 && !m.keepInvalid {
		if len(messages) < 4 || !strings.Contains(messages[len(messages)-1].Content, "exact part ID") {
			return assistant.Response{}, errors.New("missing evidence repair feedback")
		}
		arguments = `{"candidates":[{"proposal":{"action":"create_item","title":"Setup appointment","kind":"event"},"question":"Which date?","evidence":[{"part":"1_2","excerpt":"SUMMARY:Setup appointment"},{"part":"1_2","excerpt":"END:VEVENT"}]}],"questions":[]}`
	}
	return assistant.Response{Calls: []assistant.Call{{Name: "extract_capture", Arguments: arguments}}, CostUSD: 0.001}, nil
}

func TestCaptureRepairsEvidenceWithoutRelaxingValidation(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Transact(func(tx port.Transaction) error {
		return tx.PutCapture(domain.Capture{ID: "repair-source", Status: "processing", Revision: 1})
	}); err != nil {
		t.Fatal(err)
	}
	model := &repairEvidenceModel{}
	svc := assistant.New(db, command.New(db), model, 5, 1)
	result, err := svc.Extract(context.Background(), domain.CaptureDocument{CaptureID: "repair-source", Parts: []domain.CapturePart{{ID: "1_2", Text: "SUMMARY:Setup appointment\nEND:VEVENT"}}}, nil)
	if err != nil || len(result.Candidates) != 1 || model.calls != 2 {
		t.Fatal("expected one corrected extraction", result, model.calls, err)
	}
	capture, _ := db.Capture("repair-source")
	cost, _ := db.MonthCost()
	items, _ := db.Items()
	if capture.Calls != 2 || cost != 0.002 || len(items) != 0 {
		t.Fatal("repair must count both calls and preserve review", capture.Calls, cost, len(items))
	}
}

func TestCaptureEvidenceRepairStopsAtBudgets(t *testing.T) {
	for _, initialCalls := range []int{0, 9} {
		t.Run(fmt.Sprint(initialCalls), func(t *testing.T) {
			db, err := store.Open(filepath.Join(t.TempDir(), "db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := db.Transact(func(tx port.Transaction) error {
				return tx.PutCapture(domain.Capture{ID: "bounded-repair", Status: "processing", Revision: 1, Calls: initialCalls})
			}); err != nil {
				t.Fatal(err)
			}
			model := &repairEvidenceModel{keepInvalid: true}
			svc := assistant.New(db, command.New(db), model, 5, 1)
			_, err = svc.Extract(context.Background(), domain.CaptureDocument{CaptureID: "bounded-repair", Parts: []domain.CapturePart{{ID: "1_2", Text: "SUMMARY:Setup appointment\nEND:VEVENT"}}}, nil)
			if err == nil {
				t.Fatal("invalid evidence accepted")
			}
			wantCalls := 2
			if initialCalls == 9 {
				wantCalls = 1
				var invalid *domain.Invalid
				if !errors.As(err, &invalid) {
					t.Fatal(err)
				}
			}
			capture, _ := db.Capture("bounded-repair")
			cost, _ := db.MonthCost()
			if model.calls != wantCalls || capture.Calls != initialCalls+wantCalls || cost != float64(wantCalls)*0.001 {
				t.Fatal("repair exceeded its budget", model.calls, capture.Calls, cost)
			}
		})
	}
}
