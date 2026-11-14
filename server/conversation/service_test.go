package conversation_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/conversation"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
)

func openDatabase(t *testing.T) *store.Store {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// heldAgent blocks in Run until released or cancelled and reports each stage.
type heldAgent struct{ started, cancelled, release chan struct{} }

func (a heldAgent) Run(ctx context.Context, _ domain.AgentTurn) (domain.AgentOutcome, error) {
	close(a.started)
	<-ctx.Done()
	close(a.cancelled)
	<-a.release
	return domain.AgentOutcome{}, ctx.Err()
}

func TestCloseCancelsAndWaitsForTheActiveTurnThenRestartResumesIt(t *testing.T) {
	database := openDatabase(t)
	agent := heldAgent{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	finished := &recorder{}
	service := conversation.New(context.Background(), database, agent, finished)
	accepted, err := service.Accept("  Hello  ")
	if err != nil || accepted.Entry.Text != "Hello" {
		t.Fatal(accepted, err)
	}
	<-agent.started
	if _, err = service.Accept("Second"); !errors.Is(err, domain.ErrBusy) {
		t.Fatalf("want busy, got %v", err)
	}
	closed := make(chan struct{})
	go func() { service.Close(); close(closed) }()
	select {
	case <-agent.cancelled:
	case <-time.After(time.Second):
		t.Fatal("active turn not cancelled")
	}
	select {
	case <-closed:
		t.Fatal("close returned before the turn finished")
	default:
	}
	close(agent.release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close did not finish")
	}
	if _, err = service.Accept("After close"); err == nil {
		t.Fatal("accepted after close")
	}
	if commits, failures := finished.counts(); commits != 0 || failures != 0 {
		t.Fatalf("a turn cut short by shutdown must stay queued: %d commits %d failures", commits, failures)
	}
	resumed := &recorder{}
	again := conversation.New(context.Background(), database, scripted{}, resumed)
	defer again.Close()
	waitFor(t, func() bool { c, _ := resumed.counts(); return c == 1 })
	if resumed.committed[0].ID != accepted.ReplyID {
		t.Fatalf("resumed %q, want %q", resumed.committed[0].ID, accepted.ReplyID)
	}
}

func waitFor(t *testing.T, done func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if done() {
			return
		}
	}
	t.Fatal("condition never held")
}

// gatedAgent holds the first turn until released and records the order turns ran in.
type gatedAgent struct {
	release chan struct{}
	mu      sync.Mutex
	order   []string
}

func (a *gatedAgent) Run(_ context.Context, turn domain.AgentTurn) (domain.AgentOutcome, error) {
	a.mu.Lock()
	first := len(a.order) == 0
	a.order = append(a.order, turn.ID)
	a.mu.Unlock()
	if first {
		<-a.release
	}
	return domain.AgentOutcome{}, nil
}

func TestASourceArrivingDuringAnOwnerTurnWaitsItsTurnAndIsNotDropped(t *testing.T) {
	database := openDatabase(t)
	agent := &gatedAgent{release: make(chan struct{})}
	finished := &recorder{db: database}
	service := conversation.New(context.Background(), database, agent, finished)
	defer service.Close()
	accepted, _ := service.Accept("Question")
	waitFor(t, func() bool { agent.mu.Lock(); defer agent.mu.Unlock(); return len(agent.order) == 1 })
	if err := database.Transact(func(tx port.Transaction) error { return tx.PutAgentTurn(domain.ArrivalTurn("mail", 1)) }); err != nil {
		t.Fatal(err)
	}
	time.Sleep(700 * time.Millisecond)
	agent.mu.Lock()
	ran := len(agent.order)
	agent.mu.Unlock()
	if ran != 1 {
		t.Fatalf("the arrival ran while the owner turn was active: %d", ran)
	}
	close(agent.release)
	waitFor(t, func() bool { c, _ := finished.counts(); return c == 2 })
	if got := []string{agent.order[0], agent.order[1]}; got[0] != accepted.ReplyID || got[1] != "arrival-mail-1" {
		t.Fatalf("order %v", got)
	}
}

func TestLongTextWithinTheSourceBudgetReachesTheStoreIntact(t *testing.T) {
	for name, text := range map[string]string{
		"ascii":     strings.Repeat("a", conversation.MaxMessageBytes),
		"multibyte": strings.Repeat("é", conversation.MaxMessageBytes/2),
	} {
		t.Run(name, func(t *testing.T) {
			database := openDatabase(t)
			service := conversation.New(context.Background(), database, scripted{}, &recorder{})
			defer service.Close()
			accepted, err := service.Accept(text)
			if err != nil || accepted.Entry.Text != text {
				t.Fatalf("%v", err)
			}
			history, _ := database.History(0, 5)
			if len(history) == 0 || history[0].Text != text {
				t.Fatal("stored text differs")
			}
		})
	}
}

func TestEmptyOversizedAndInvalidTextAreRejectedBeforeAnythingIsSaved(t *testing.T) {
	database := openDatabase(t)
	service := conversation.New(context.Background(), database, scripted{}, &recorder{})
	defer service.Close()
	for name, text := range map[string]string{
		"empty":     "  \n ",
		"too long":  strings.Repeat("a", conversation.MaxMessageBytes+1),
		"multibyte": strings.Repeat("é", conversation.MaxMessageBytes/2+1),
		"invalid":   "bad \xff byte",
	} {
		_, err := service.Accept(text)
		var invalid *domain.Invalid
		if !errors.As(err, &invalid) || invalid.Field != "text" {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if history, _ := database.History(0, 5); len(history) != 0 {
		t.Fatalf("rejected text was saved: %d entries", len(history))
	}
}

func TestACommitFailureFailsTheTurnOnce(t *testing.T) {
	finished := &recorder{fail: errors.New("disk full")}
	service := conversation.New(context.Background(), openDatabase(t), scripted{}, finished)
	defer service.Close()
	accepted, _ := service.Accept("Hello")
	waitFor(t, func() bool { _, f := finished.counts(); return f == 1 })
	if finished.failed[0].ID != accepted.ReplyID || finished.causes[0].Error() != "disk full" {
		t.Fatalf("%+v %v", finished.failed, finished.causes)
	}
	time.Sleep(100 * time.Millisecond)
	if _, f := finished.counts(); f != 1 {
		t.Fatalf("turn failed %d times", f)
	}
}
