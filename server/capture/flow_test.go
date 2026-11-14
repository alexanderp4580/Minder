package capture_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/capture"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/mailparse"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
)

// threeAppointments extracts three events; "a" and "b" ask how long they take until answered.
type threeAppointments struct{ calls int }

func (x *threeAppointments) Extract(_ context.Context, _ domain.CaptureDocument, answers []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	x.calls++
	evidence := []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}
	event := func(id, title, question string) domain.CaptureCandidate {
		return domain.CaptureCandidate{ID: id, Question: question, Evidence: evidence, Proposal: domain.Proposal{Action: "create_item", Title: title, Kind: "event", StartAt: "2026-11-20T10:00:00+01:00"}}
	}
	a, b := event("a", "Dentist", "How long will it take?"), event("b", "Checkup", "How long will it take?")
	for _, answer := range answers {
		if answer.CandidateID == "a" {
			a.Question = ""
		}
		if answer.CandidateID == "b" {
			b.Question = ""
		}
	}
	return domain.CaptureExtraction{Summary: "Three visits await your review.", Notification: "Three dental visits need a decision.", Candidates: []domain.CaptureCandidate{a, b, event("c", "Follow-up", "")}}, nil
}

func flowFixture(t *testing.T, x capture.Extractor) (*capture.Service, *store.Store, *command.Service, string) {
	t.Helper()
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, x)
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, e := svc.Ingest(context.Background(), domain.CaptureEnvelope{From: "owner@example.com", RequestID: "flow"}, []byte("Subject: Checkups\r\nFrom: owner@example.com\r\n\r\nHotel"))
	if e != nil {
		t.Fatal(e)
	}
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	return svc, db, command.New(db), id
}

func stateOf(t *testing.T, svc *capture.Service, id string) string {
	t.Helper()
	c, e := svc.Get(id)
	if e != nil {
		t.Fatal(e)
	}
	var out []string
	for _, candidate := range c.Candidates {
		out = append(out, candidate.ID+"="+candidate.State)
	}
	return strings.Join(out, " ")
}

func chatTexts(t *testing.T, db *store.Store) []string {
	t.Helper()
	history, _ := db.History(0, 100)
	out := []string{}
	for _, entry := range history {
		if entry.Kind == "minder" || entry.Kind == "question" {
			out = append(out, entry.Text)
		}
	}
	return out
}

func changeOf(t *testing.T, svc *capture.Service, id, candidate string) string {
	t.Helper()
	c, _ := svc.Get(id)
	for _, cand := range c.Candidates {
		if cand.ID == candidate {
			return cand.ChangeID
		}
	}
	return ""
}

func TestExtractionPublishesOnlyTheFirstCandidateAndSaysWhatItFound(t *testing.T) {
	svc, db, _, id := flowFixture(t, &threeAppointments{})
	if got := stateOf(t, svc, id); got != "a=needs-answer b=queued c=queued" {
		t.Fatalf("states %s", got)
	}
	texts := chatTexts(t, db)
	if len(texts) < 2 || texts[0] != "Three visits await your review." || texts[1] != "Appointment 1 of 3: Dentist" {
		t.Fatalf("chat %q", texts)
	}
}

func TestOnlyTheCurrentCandidateAsksItsQuestion(t *testing.T) {
	svc, db, _, id := flowFixture(t, &threeAppointments{})
	history, _ := db.History(0, 100)
	for _, entry := range history {
		if entry.Kind != "question" {
			continue
		}
		asking := 0
		for _, candidate := range entry.Capture.Candidates {
			if candidate.State == "needs-answer" {
				asking++
			}
		}
		if asking != 1 {
			t.Fatalf("a question entry shows %d open questions", asking)
		}
	}
	if changeOf(t, svc, id, "b") != "" {
		t.Fatal("a queued candidate has a preview")
	}
}

func TestAnsweringTheCurrentQuestionKeepsTheQueuedOnes(t *testing.T) {
	x := &threeAppointments{}
	svc, _, _, id := flowFixture(t, x)
	c, _ := svc.Get(id)
	if _, e := svc.AnswerCandidate(context.Background(), id, c.Revision, "a", "1 hour"); e != nil {
		t.Fatal(e)
	}
	if e := svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	if got := stateOf(t, svc, id); got != "a=pending b=queued c=queued" {
		t.Fatalf("states %s", got)
	}
}

func TestConfirmingOrDiscardingPublishesTheNext(t *testing.T) {
	svc, db, commands, id := flowFixture(t, &threeAppointments{})
	c, _ := svc.Get(id)
	svc.AnswerCandidate(context.Background(), id, c.Revision, "a", "1 hour")
	svc.ProcessNext(context.Background(), time.Now())
	if _, e := commands.Confirm(changeOf(t, svc, id, "a")); e != nil {
		t.Fatal(e)
	}
	if got := stateOf(t, svc, id); got != "a=confirmed b=needs-answer c=queued" {
		t.Fatalf("after confirm: %s", got)
	}
	c, _ = svc.Get(id)
	svc.AnswerCandidate(context.Background(), id, c.Revision, "b", "30 minutes")
	svc.ProcessNext(context.Background(), time.Now())
	if _, e := commands.Discard(changeOf(t, svc, id, "b")); e != nil {
		t.Fatal(e)
	}
	if got := stateOf(t, svc, id); got != "a=confirmed b=discarded c=pending" {
		t.Fatalf("after discard: %s", got)
	}
	if _, e := commands.Confirm(changeOf(t, svc, id, "c")); e != nil {
		t.Fatal(e)
	}
	texts := chatTexts(t, db)
	if texts[len(texts)-1] != "Done: 2 added, 1 skipped." {
		t.Fatalf("the review did not end with a summary: %q", texts)
	}
	if got := strings.Join(texts, "|"); !strings.Contains(got, "Appointment 2 of 3: Checkup") || !strings.Contains(got, "Appointment 3 of 3: Follow-up") {
		t.Fatalf("progress messages missing: %q", texts)
	}
}
