package assistant_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

type captureModel struct{ messages []assistant.Message }

func (m *captureModel) Complete(_ context.Context, messages []assistant.Message) (assistant.Response, error) {
	m.messages = messages
	return assistant.Response{Calls: []assistant.Call{{Name: "extract_capture", Arguments: `{"summary":"One visit to review.","notification":"Please review the visit.","candidates":[{"proposal":{"action":"create_item","title":"Appointment","kind":"event"},"question":"What date is the appointment?","evidence":[{"part":"1","excerpt":"appointment"}]}],"questions":[]}`}}, CostUSD: 0.001}, nil
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
	if result.Summary != "One visit to review." || result.Notification != "Please review the visit." {
		t.Fatalf("model text lost: %+v", result)
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

func TestCaptureEvidenceRepairIsOneCorrectionWithoutACallCap(t *testing.T) {
	for _, initialCalls := range []int{0, 9, 50} {
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
			var invalid *domain.Invalid
			if errors.As(err, &invalid) {
				t.Fatalf("the extraction was refused by a call cap: %v", err)
			}
			capture, _ := db.Capture("bounded-repair")
			cost, _ := db.MonthCost()
			if model.calls != 2 || capture.Calls != initialCalls+2 || cost != 0.002 {
				t.Fatal("expected the original call and one correction", model.calls, capture.Calls, cost)
			}
		})
	}
}

func TestAnEmailOfOneHundredKilobytesIsAcceptedAndOverFourHundredIsRefused(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := assistant.New(db, command.New(db), &captureModel{}, 5, 1)
	big := func(n int) domain.CaptureDocument {
		return domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1", Text: "appointment " + strings.Repeat("x", n)}}}
	}
	if _, err := svc.Extract(context.Background(), big(100000), nil); err != nil {
		t.Fatalf("a 100 KB email was refused: %v", err)
	}
	var invalid *domain.Invalid
	if _, err := svc.Extract(context.Background(), big(450000), nil); !errors.As(err, &invalid) {
		t.Fatalf("a 450 KB email was accepted: %v", err)
	}
}

type structuredModel struct{ messages []assistant.Message }

func (m *structuredModel) Complete(_ context.Context, messages []assistant.Message) (assistant.Response, error) {
	m.messages = messages
	return assistant.Response{Calls: []assistant.Call{{Name: "extract_capture", Arguments: `{"candidates":[{"id":"fit","proposal":{"action":"create_item","title":"Fitting","kind":"event","start_at":"2026-11-20T14:30:00+01:00","reminder_offsets_minutes":[60,7,1440,99999]},"question":"How long will it take?","options":["30 minutes","45 minutes","1 hour","2 hours"],"evidence":[{"part":"1","excerpt":"Fitting"}]}],"questions":[{"text":"Is the location important?","options":["Skip","","Use the email address","Other","Fourth"]},"Who attends?"]}`}}, CostUSD: 0.001}, nil
}

func TestExtractionNormalisesQuestionOptionsAndOffsets(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	m := &structuredModel{}
	svc := assistant.New(db, command.New(db), m, 5, 1)
	result, e := svc.Extract(context.Background(), domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1", Text: "Fitting"}}}, nil)
	if e != nil || len(result.Candidates) != 1 || len(result.Questions) != 2 {
		t.Fatal(result, e)
	}
	candidate := result.Candidates[0]
	if len(candidate.Options) != 3 || candidate.Options[2] != "1 hour" {
		t.Fatal("candidate options must cap at three", candidate.Options)
	}
	if got := candidate.Proposal.ReminderOffsets; len(got) != 2 || got[0] != 1440 || got[1] != 60 {
		t.Fatal("invalid offsets must be dropped and the rest kept longest first", got)
	}
	first := result.Questions[0]
	if len(first.Options) != 3 || first.Options[0] != "Skip" || first.Options[1] != "Use the email address" || first.ID == "" {
		t.Fatal("question options", first)
	}
	if result.Questions[1].Text != "Who attends?" || result.Questions[1].ID == first.ID {
		t.Fatal("bare string question or ids", result.Questions)
	}
}

func TestPromptAsksLikeAnAssistantNotFromAChecklist(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	m := &structuredModel{}
	svc := assistant.New(db, command.New(db), m, 5, 1)
	_, _ = svc.Extract(context.Background(), domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1", Text: "Fitting"}}}, nil)
	prompt := m.messages[0].Content
	for _, want := range []string{"thoughtful assistant", "quick replies", "reminder_offsets_minutes", "location", "short list"} {
		if !strings.Contains(prompt, want) {
			t.Fatal("prompt lacks", want)
		}
	}
	if strings.Contains(prompt, "needs a location") {
		t.Fatal("prompt hard-codes a required field")
	}
}

func TestRejectedExtractionsLogReasonsNotContent(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := assistant.New(db, command.New(db), &repairEvidenceModel{keepInvalid: true}, 5, 1)
	_, err = svc.Extract(context.Background(), domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1_2", Text: "SUMMARY:Setup appointment\nEND:VEVENT"}}}, nil)
	if err == nil {
		t.Fatal("invalid evidence accepted")
	}
	attempts := map[float64]bool{}
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var record map[string]any
		if json.Unmarshal([]byte(line), &record) == nil && record["msg"] == "email extraction rejected" {
			if record["reason"] != "evidence" {
				t.Fatalf("reason %v", record["reason"])
			}
			attempts[record["attempt"].(float64)] = true
		}
	}
	if !attempts[1] || !attempts[2] {
		t.Fatalf("attempts logged %v", attempts)
	}
	for _, private := range []string{"Setup appointment", "SUMMARY", "VEVENT"} {
		if strings.Contains(buf.String(), private) {
			t.Fatalf("email content %q in logs: %s", private, buf.String())
		}
	}
}

// partialModel returns two candidates, one of which quotes text that is not in the email.
// When fixOnSecond is set, its second answer quotes real text for both.
type partialModel struct {
	calls        int
	fixOnSecond  bool
	failOnSecond bool
}

func (m *partialModel) Complete(context.Context, []assistant.Message) (assistant.Response, error) {
	m.calls++
	if m.failOnSecond && m.calls == 2 {
		return assistant.Response{}, &domain.ModelError{Message: "Provider unavailable"}
	}
	second := "this quote is not in the email"
	if m.fixOnSecond && m.calls == 2 {
		second = "Checkup"
	}
	arguments := `{"candidates":[
{"id":"a","proposal":{"action":"create_item","title":"Dentist","kind":"event","start_at":"2026-11-20T10:00:00+01:00"},"evidence":[{"part":"1","excerpt":"Dentist"}]},
{"id":"b","proposal":{"action":"create_item","title":"Checkup","kind":"event","start_at":"2026-11-21T10:00:00+01:00"},"evidence":[{"part":"1","excerpt":"` + second + `"}]}],"questions":[]}`
	return assistant.Response{Calls: []assistant.Call{{Name: "extract_capture", Arguments: arguments}}, CostUSD: 0.001}, nil
}

func extractFrom(t *testing.T, m assistant.Model) (domain.CaptureExtraction, error) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	svc := assistant.New(db, command.New(db), m, 5, 1)
	return svc.Extract(context.Background(), domain.CaptureDocument{Parts: []domain.CapturePart{{ID: "1", Text: "Dentist on Friday. Checkup on Saturday."}}}, nil)
}

func TestOneBadQuoteKeepsTheValidCandidates(t *testing.T) {
	m := &partialModel{}
	result, err := extractFrom(t, m)
	if err != nil || len(result.Candidates) != 1 || result.Candidates[0].ID != "a" || result.Dropped != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if m.calls != 2 {
		t.Fatalf("a partial answer gets one correction, got %d calls", m.calls)
	}
}

func TestACorrectionThatFixesEverythingDropsNothing(t *testing.T) {
	m := &partialModel{fixOnSecond: true}
	result, err := extractFrom(t, m)
	if err != nil || len(result.Candidates) != 2 || result.Dropped != 0 || m.calls != 2 {
		t.Fatalf("%+v %v calls %d", result, err, m.calls)
	}
}

func TestNoValidCandidateFailsWithTheEvidenceReason(t *testing.T) {
	_, err := extractFrom(t, &repairEvidenceModel{keepInvalid: true})
	if got := domain.ReasonOf(err); got != domain.ReasonEvidence {
		t.Fatalf("reason %s: %v", got, err)
	}
}

func TestTheModelDecidesTheLocation(t *testing.T) {
	for _, location := range []string{"", "Microsoft Teams", "Kerkstraat 1, Utrecht"} {
		result, err := extractFrom(t, &locatedModel{location: location})
		if err != nil || len(result.Candidates) != 1 || result.Candidates[0].Proposal.Location != location {
			t.Fatalf("%q became %q (%v)", location, result.Candidates[0].Proposal.Location, err)
		}
	}
}

type locatedModel struct{ location string }

func (m *locatedModel) Complete(context.Context, []assistant.Message) (assistant.Response, error) {
	arguments := `{"candidates":[{"id":"a","proposal":{"action":"create_item","title":"Dentist","kind":"event","location":"` + m.location + `","start_at":"2026-11-20T10:00:00+01:00"},"evidence":[{"part":"1","excerpt":"Dentist"}]}],"questions":[]}`
	return assistant.Response{Calls: []assistant.Call{{Name: "extract_capture", Arguments: arguments}}, CostUSD: 0.001}, nil
}

func TestACorrectionProviderFailureKeepsVerifiedCandidates(t *testing.T) {
	result, err := extractFrom(t, &partialModel{failOnSecond: true})
	if err != nil || len(result.Candidates) != 1 || result.Candidates[0].ID != "a" || result.Dropped != 1 {
		t.Fatalf("verified appointment lost: %+v %v", result, err)
	}
}
