package conversation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/conversation"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
)

// openReview stores a review with one open question and one queued candidate.
func openReview(t *testing.T, db *store.Store) {
	t.Helper()
	err := db.Transact(func(tx port.Transaction) error {
		return tx.PutCapture(domain.Capture{ID: "mail", Revision: 3, Status: "needs-answer", Candidates: []domain.CaptureCandidate{
			{ID: "a", Revision: 3, State: "needs-answer"}, {ID: "b", Revision: 3, State: "queued"}, {ID: "old", Revision: 1, State: "confirmed"}}})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// claimed runs the service until the agent has seen one turn and returns it.
type seen struct{ turns chan domain.AgentTurn }

func (s seen) Run(_ context.Context, turn domain.AgentTurn) (domain.AgentOutcome, error) {
	s.turns <- turn
	return domain.AgentOutcome{}, nil
}

func TestAQuickReplyQueuesAnAgentTurnAndChangesNoCandidate(t *testing.T) {
	db := openDatabase(t)
	openReview(t, db)
	agent := seen{make(chan domain.AgentTurn, 1)}
	service := conversation.New(context.Background(), db, agent, &recorder{db: db})
	defer service.Close()
	entry, err := service.AnswerReview("mail", 3, "a", "  1 hour ")
	if err != nil || entry.Kind != "user" || entry.Text != "1 hour" {
		t.Fatalf("%+v %v", entry, err)
	}
	select {
	case turn := <-agent.turns:
		if turn.Kind != domain.TurnReviewReply || turn.SourceID != "mail" || turn.CandidateID != "a" || turn.ExpectedRevision != 3 || turn.EntryID != entry.ID {
			t.Fatalf("%+v", turn)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no turn ran")
	}
	c, _ := db.Capture("mail")
	if c.Revision != 3 || c.Candidates[0].State != "needs-answer" || c.Status != "needs-answer" {
		t.Fatalf("an unanswered quick reply changed the review: %+v", c)
	}
}

func TestAQuickReplyForAStaleOrClosedReviewIsRefusedBeforeSaving(t *testing.T) {
	db := openDatabase(t)
	openReview(t, db)
	service := conversation.New(context.Background(), db, scripted{}, &recorder{})
	defer service.Close()
	for name, call := range map[string]func() error{
		"old revision":       func() error { _, e := service.AnswerReview("mail", 2, "a", "1 hour"); return e },
		"confirmed":          func() error { _, e := service.AnswerReview("mail", 3, "old", "1 hour"); return e },
		"unknown candidate":  func() error { _, e := service.AnswerReview("mail", 3, "zzz", "1 hour"); return e },
		"queued, not asking": func() error { _, e := service.AnswerReview("mail", 3, "b", "1 hour"); return e },
	} {
		if err := call(); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := service.AnswerReview("missing", 1, "a", "1 hour"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	var invalid *domain.Invalid
	if _, err := service.AnswerReview("mail", 3, "a", "  "); !errors.As(err, &invalid) {
		t.Fatal(err)
	}
	if history, _ := db.History(0, 10); len(history) != 0 {
		t.Fatalf("a refused answer was saved: %d entries", len(history))
	}
}

func TestADeletedReviewRefusesAnswers(t *testing.T) {
	db := openDatabase(t)
	db.Transact(func(tx port.Transaction) error {
		return tx.PutCapture(domain.Capture{ID: "mail", Revision: 3, Deleted: true, Candidates: []domain.CaptureCandidate{{ID: "a", State: "needs-answer"}}})
	})
	service := conversation.New(context.Background(), db, scripted{}, &recorder{})
	defer service.Close()
	if _, err := service.AnswerReview("mail", 3, "a", "yes"); !errors.Is(err, domain.ErrConflict) {
		t.Fatal(err)
	}
}

func TestOnlyOneOwnerTurnWaitsAtATimeEvenForQuickReplies(t *testing.T) {
	db := openDatabase(t)
	openReview(t, db)
	release := make(chan struct{})
	service := conversation.New(context.Background(), db, blocking{release}, &recorder{db: db})
	defer service.Close()
	defer close(release)
	if _, err := service.Accept("Hello"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AnswerReview("mail", 3, "a", "1 hour"); !errors.Is(err, domain.ErrBusy) {
		t.Fatalf("%v", err)
	}
}
