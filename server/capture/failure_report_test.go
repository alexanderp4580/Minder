package capture_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
)

// missingAnswer keeps the four original items but omits the answered item from its next response.
type missingAnswer struct{}

func (missingAnswer) Extract(_ context.Context, _ domain.CaptureDocument, answers []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	evidence := []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}
	event := func(id, question string) domain.CaptureCandidate {
		return domain.CaptureCandidate{ID: id, Question: question, Evidence: evidence, Proposal: domain.Proposal{Action: "create_item", Kind: "event", Title: id, StartAt: "2026-11-20T10:00:00+01:00"}}
	}
	a, b, c := event("a", ""), event("b", "How long will it take?"), event("c", "How long will it take?")
	a.Proposal.DurationMinutes = 30
	task := domain.CaptureCandidate{ID: "d", Evidence: evidence, Proposal: domain.Proposal{Action: "create_item", Kind: "task", Title: "Questionnaire"}}
	if len(answers) > 0 {
		return domain.CaptureExtraction{Candidates: []domain.CaptureCandidate{c, task}}, nil
	}
	return domain.CaptureExtraction{Candidates: []domain.CaptureCandidate{a, b, c, task}}, nil
}

func TestAFailedAnswerPostsANewFailureMessageAndPushesOnce(t *testing.T) {
	svc, db, commands, id := flowFixture(t, missingAnswer{})
	if err := db.RegisterDevice("phone", "token"); err != nil {
		t.Fatal(err)
	}
	sender := &pushes{}
	svc.Sender = sender
	if _, err := commands.Confirm(changeOf(t, svc, id, "a")); err != nil {
		t.Fatal(err)
	}
	c, _ := svc.Get(id)
	if _, err := svc.AnswerCandidate(context.Background(), id, c.Revision, "b", "1 hour"); err != nil {
		t.Fatal(err)
	}
	err := svc.ProcessNext(context.Background(), time.Now())
	var model *domain.ModelError
	if !errors.As(err, &model) {
		t.Fatalf("expected missing-item failure, got %v", err)
	}
	c, _ = svc.Get(id)
	if c.Status != "failed" || len(c.Candidates) != 4 || c.Candidates[0].State != "confirmed" {
		t.Fatalf("failure lost review state: %+v", c)
	}
	history, _ := db.History(0, 100)
	last := history[len(history)-1]
	if last.Kind != "minder" || last.CaptureID != id || last.Text != c.Error {
		t.Fatalf("newest chat entry must report the failure, got kind=%q text=%q", last.Kind, last.Text)
	}
	if string(model.Reason) != "review_item_missing" {
		t.Fatalf("missing-item failure reason %q", model.Reason)
	}
	deliver(svc)
	deliver(svc)
	if len(sender.bodies) != 1 || sender.bodies[0] != "Minder could not read an email." {
		t.Fatalf("failure push: %q", sender.bodies)
	}
}

func TestEachFailedRevisionGetsOneChatReport(t *testing.T) {
	svc, db, id := failingService(t, &flaky{fails: 100, err: errors.New("provider down")})
	for n := 0; n < 2; n++ {
		if n > 0 {
			if _, err := svc.Retry(id); err != nil {
				t.Fatal(err)
			}
		}
		if err := svc.ProcessNext(context.Background(), time.Now()); err == nil {
			t.Fatal("expected failure")
		}
		for attempt := 0; attempt < 3; attempt++ {
			svc.ProcessNext(context.Background(), time.Now().Add(time.Hour))
		}
	}
	history, _ := db.History(0, 100)
	var reports []domain.Entry
	for _, e := range history {
		if e.Kind == "minder" && e.CaptureID == id {
			reports = append(reports, e)
		}
	}
	if len(reports) != 2 || reports[0].ID == reports[1].ID {
		t.Fatalf("want one report per failed revision, got %+v", reports)
	}
}

func TestStartupReportsAnEarlierFailureOnceWithoutReextractingOrPushingAgain(t *testing.T) {
	extractor := &flaky{fails: 100, err: errors.New("unused")}
	svc, db, id := failingService(t, extractor)
	sender := &pushes{}
	svc.Sender = sender
	if err := db.Transact(func(tx port.Transaction) error {
		c, err := tx.Capture(id)
		if err != nil {
			return err
		}
		c.Status = "failed"
		c.Error = domain.ReasonEvidence.Sentence()
		if err = tx.PutCapture(c); err != nil {
			return err
		}
		_, err = tx.PutEntry(domain.Entry{ID: c.EntryID, Kind: "capture", Text: c.Error, Capture: &c})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	deliver(svc)
	history, _ := db.History(0, 100)
	first := history[len(history)-1]
	if first.Kind != "minder" || first.CaptureID != id || first.Text != domain.ReasonEvidence.Sentence() {
		t.Fatalf("missing recovery report: %+v", first)
	}
	deliver(svc)
	history, _ = db.History(0, 100)
	last := history[len(history)-1]
	if last.ID != first.ID || last.Cursor != first.Cursor {
		t.Fatal("restart repeated or moved an existing failure report")
	}
	if extractor.calls != 0 || len(sender.bodies) != 0 {
		t.Fatalf("recovery re-extracted or re-pushed: calls=%d pushes=%d", extractor.calls, len(sender.bodies))
	}
}
