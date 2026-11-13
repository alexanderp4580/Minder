package capture_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/capture"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/mailparse"
	"github.com/alexanderp4580/Minder/server/store"
)

type manyQuestions struct{}

func (manyQuestions) Extract(context.Context, domain.CaptureDocument, []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	evidence := []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}
	return domain.CaptureExtraction{
		Candidates: []domain.CaptureCandidate{
			{ID: "a", Proposal: domain.Proposal{Action: "create_item", Title: "Fitting", Kind: "event"}, Question: "How long will it take?", Options: []string{"30 minutes", "1 hour"}, Evidence: evidence},
			{ID: "b", Proposal: domain.Proposal{Action: "create_item", Title: "Collection", Kind: "event"}, Question: "What date?", Evidence: evidence},
		},
		Questions: []domain.CaptureQuestion{{ID: "q1", Text: "Is the location important?", Options: []string{"Skip"}}},
	}, nil
}

func questionEntries(t *testing.T, db *store.Store, kind string) (found []domain.Entry, all []domain.Entry) {
	t.Helper()
	all, err := db.History(0, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range all {
		if e.Kind == kind {
			found = append(found, e)
		}
	}
	return
}

func processed(t *testing.T, extractor capture.Extractor) (*capture.Service, *store.Store, string) {
	t.Helper()
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, extractor)
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, e := svc.Ingest(context.Background(), domain.CaptureEnvelope{RequestID: "q"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e != nil {
		t.Fatal(e)
	}
	if e = svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	return svc, db, id
}

func TestQuestionsPostAsSeparateEntryAfterReceived(t *testing.T) {
	_, db, id := processed(t, questionsExtractor{})
	questions, all := questionEntries(t, db, "question")
	if len(questions) != 1 || questions[0].CaptureID != id || questions[0].Capture == nil {
		t.Fatal("expected one question entry", questions)
	}
	cardIndex, questionIndex := -1, -1
	for n, e := range all {
		if e.Kind == "capture" {
			cardIndex = n
		}
		if e.Kind == "question" {
			questionIndex = n
		}
	}
	if cardIndex < 0 || questionIndex <= cardIndex {
		t.Fatal("the question must follow the email card", cardIndex, questionIndex)
	}
}

func TestNoQuestionEntryWhenNothingMissing(t *testing.T) {
	_, db, _ := processed(t, extracted{})
	if questions, _ := questionEntries(t, db, "question"); len(questions) != 0 {
		t.Fatal("a complete email must not ask anything", questions)
	}
}

func TestSeveralQuestionsShareOneEntry(t *testing.T) {
	_, db, _ := processed(t, manyQuestions{})
	questions, _ := questionEntries(t, db, "question")
	if len(questions) != 1 {
		t.Fatal("expected one entry for all questions", len(questions))
	}
	c := questions[0].Capture
	if len(c.Questions) != 1 || c.Questions[0].Options[0] != "Skip" || c.Candidates[0].Options[1] != "1 hour" {
		t.Fatal("options lost", c)
	}
}

func TestAnswerClosesThePreviousQuestionEntry(t *testing.T) {
	svc, db, id := processed(t, questionsExtractor{})
	c, _ := svc.Get(id)
	if _, err := svc.Answer(context.Background(), id, c.Revision, "21 November"); err != nil {
		t.Fatal(err)
	}
	if open, _ := questionEntries(t, db, "question"); len(open) != 0 {
		t.Fatal("the answered question stayed open", open)
	}
	closed, _ := questionEntries(t, db, "minder")
	if len(closed) != 1 || closed[0].Text != "Questions answered." {
		t.Fatal(closed)
	}
}

func TestDismissClosesTheQuestionEntry(t *testing.T) {
	svc, db, id := processed(t, questionsExtractor{})
	c, _ := svc.Get(id)
	if _, err := svc.Dismiss(context.Background(), id, c.Revision); err != nil {
		t.Fatal(err)
	}
	if open, _ := questionEntries(t, db, "question"); len(open) != 0 {
		t.Fatal("dismissed capture kept an open question", open)
	}
}
