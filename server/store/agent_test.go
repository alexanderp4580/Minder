package store_test

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func ownerTurn(id string) domain.AgentTurn {
	return domain.AgentTurn{ID: id, Kind: domain.TurnOwner}
}

func TestEnqueueStoresTheOwnerEntryAndTheQueuedTurnTogether(t *testing.T) {
	s := openStore(t)
	turn, entry, err := s.EnqueueAgentTurn(ownerTurn("t1"), &domain.Entry{Kind: "user", Text: "Hello"})
	if err != nil || turn.State != domain.TurnQueued || turn.EntryID == "" || turn.EntryID != entry.ID || entry.Cursor == 0 {
		t.Fatalf("%+v %+v %v", turn, entry, err)
	}
	history, _ := s.History(0, 10)
	if len(history) != 1 || history[0].Text != "Hello" {
		t.Fatalf("%+v", history)
	}
}

func TestEnqueueWithoutAnEntryQueuesOnlyTheTurn(t *testing.T) {
	s := openStore(t)
	turn, entry, err := s.EnqueueAgentTurn(domain.ArrivalTurn("mail", 1), nil)
	if err != nil || turn.State != domain.TurnQueued || entry.ID != "" {
		t.Fatalf("%+v %+v %v", turn, entry, err)
	}
	if history, _ := s.History(0, 10); len(history) != 0 {
		t.Fatalf("%+v", history)
	}
}

func TestEnqueueingTheSameTurnTwiceKeepsOneTurnAndOneEntry(t *testing.T) {
	s := openStore(t)
	first, _, _ := s.EnqueueAgentTurn(ownerTurn("t1"), &domain.Entry{Kind: "user", Text: "Hello"})
	again, entry, err := s.EnqueueAgentTurn(ownerTurn("t1"), &domain.Entry{Kind: "user", Text: "Hello"})
	if err != nil || again.EntryID != first.EntryID || entry.ID != first.EntryID {
		t.Fatalf("%+v %+v %v", again, entry, err)
	}
	if history, _ := s.History(0, 10); len(history) != 1 {
		t.Fatalf("duplicate entry: %+v", history)
	}
}

func TestAFailedTransactionLeavesNeitherEntryNorTurn(t *testing.T) {
	s := openStore(t)
	stop := errors.New("stop")
	err := s.Transact(func(tx port.Transaction) error {
		if _, err := tx.PutEntry(domain.Entry{ID: "e", Kind: "user", Text: "Hello"}); err != nil {
			return err
		}
		if err := tx.PutAgentTurn(domain.AgentTurn{ID: "t", Kind: domain.TurnOwner, EntryID: "e", State: domain.TurnQueued}); err != nil {
			return err
		}
		return stop
	})
	if !errors.Is(err, stop) {
		t.Fatal(err)
	}
	if _, err = s.ClaimAgentTurn(time.Now()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("turn survived: %v", err)
	}
	if history, _ := s.History(0, 10); len(history) != 0 {
		t.Fatalf("entry survived: %+v", history)
	}
}

func TestClaimTakesTurnsInArrivalOrderOneAtATime(t *testing.T) {
	s := openStore(t)
	now := time.Now()
	s.EnqueueAgentTurn(ownerTurn("first"), &domain.Entry{Kind: "user", Text: "a"})
	s.EnqueueAgentTurn(domain.ArrivalTurn("mail", 1), nil)
	got, err := s.ClaimAgentTurn(now)
	if err != nil || got.ID != "first" || got.State != domain.TurnActive || got.LeaseUntil <= now.Unix() {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err = s.ClaimAgentTurn(now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("a second turn was claimed during the lease: %v", err)
	}
	got.State = domain.TurnDone
	if err = s.Transact(func(tx port.Transaction) error { return tx.PutAgentTurn(got) }); err != nil {
		t.Fatal(err)
	}
	next, err := s.ClaimAgentTurn(now)
	if err != nil || next.ID != "arrival-mail-1" {
		t.Fatalf("%+v %v", next, err)
	}
}

func TestSimultaneousClaimsGiveTheTurnToOneWorker(t *testing.T) {
	s := openStore(t)
	s.EnqueueAgentTurn(ownerTurn("a"), &domain.Entry{Kind: "user", Text: "a"})
	s.EnqueueAgentTurn(ownerTurn("b"), &domain.Entry{Kind: "user", Text: "b"})
	var wg sync.WaitGroup
	claimed := make(chan string, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if turn, err := s.ClaimAgentTurn(time.Now()); err == nil {
				claimed <- turn.ID
			}
		}()
	}
	wg.Wait()
	close(claimed)
	if got := len(claimed); got != 1 {
		t.Fatalf("%d workers claimed a turn", got)
	}
}

func TestAnExpiredLeaseLetsTheTurnRunAgain(t *testing.T) {
	s := openStore(t)
	s.EnqueueAgentTurn(ownerTurn("a"), &domain.Entry{Kind: "user", Text: "a"})
	now := time.Now()
	first, _ := s.ClaimAgentTurn(now)
	again, err := s.ClaimAgentTurn(time.Unix(first.LeaseUntil+1, 0))
	if err != nil || again.ID != "a" {
		t.Fatalf("%+v %v", again, err)
	}
}

func TestPendingOwnerWorkIgnoresSourceArrivals(t *testing.T) {
	s := openStore(t)
	s.EnqueueAgentTurn(domain.ArrivalTurn("mail", 1), nil)
	if busy, err := s.OwnerTurnPending(); err != nil || busy {
		t.Fatalf("%v %v", busy, err)
	}
	s.EnqueueAgentTurn(ownerTurn("a"), &domain.Entry{Kind: "user", Text: "a"})
	if busy, err := s.OwnerTurnPending(); err != nil || !busy {
		t.Fatalf("%v %v", busy, err)
	}
}

func TestAReleasedTurnIsClaimedAgainAtOnce(t *testing.T) {
	s := openStore(t)
	s.EnqueueAgentTurn(ownerTurn("a"), &domain.Entry{Kind: "user", Text: "a"})
	now := time.Now()
	claimed, _ := s.ClaimAgentTurn(now)
	if err := s.ReleaseAgentTurn(claimed.ID); err != nil {
		t.Fatal(err)
	}
	if again, err := s.ClaimAgentTurn(now); err != nil || again.ID != "a" {
		t.Fatalf("%+v %v", again, err)
	}
}
