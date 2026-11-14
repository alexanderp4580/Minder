package review_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/review"
	"github.com/alexanderp4580/Minder/server/store"
)

func previewing(tx port.Transaction, p domain.Proposal) (domain.Entry, error) {
	return tx.PutEntry(domain.Entry{Kind: "preview", Change: &domain.Change{ID: "change-" + p.CandidateID, State: "pending"}})
}

func candidate(id, state string) domain.CaptureCandidate {
	return domain.CaptureCandidate{ID: id, State: state, Proposal: domain.Proposal{Action: "create_item", Title: strings.ToUpper(id), Kind: "event", CandidateID: id}}
}

// advance runs review.Advance on a capture inside a real transaction and returns it with the chat texts that were written.
func advance(t *testing.T, c domain.Capture) (domain.Capture, []string) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	c.ID, c.Revision = "mail", 1
	if err = db.Transact(func(tx port.Transaction) error { return review.Advance(tx, &c, previewing) }); err != nil {
		t.Fatal(err)
	}
	history, _ := db.History(0, 50)
	texts := []string{}
	for _, entry := range history {
		texts = append(texts, entry.Kind+": "+entry.Text)
	}
	return c, texts
}

func states(c domain.Capture) string {
	var out []string
	for _, candidate := range c.Candidates {
		out = append(out, candidate.ID+"="+candidate.State)
	}
	return strings.Join(out, " ")
}

func TestOpeningCountsWhatWasFound(t *testing.T) {
	events := func(n int) []domain.CaptureCandidate {
		out := make([]domain.CaptureCandidate, n)
		for i := range out {
			out[i] = candidate("x", "queued")
		}
		return out
	}
	task := candidate("t", "queued")
	task.Proposal.Kind = ""
	for name, tc := range map[string]struct {
		subject    string
		candidates []domain.CaptureCandidate
		want       string
	}{
		"three":   {"Checkups", events(3), `I found 3 appointments in "Checkups". We will go through them one at a time.`},
		"one":     {"Checkups", events(1), `I found 1 appointment in "Checkups".`},
		"items":   {"Checkups", append(events(1), task), `I found 2 items in "Checkups". We will go through them one at a time.`},
		"no name": {"", events(2), `I found 2 appointments in your email. We will go through them one at a time.`},
	} {
		if got := review.Opening(tc.subject, tc.candidates); got != tc.want {
			t.Fatalf("%s: %q", name, got)
		}
	}
}

func TestAdvancePublishesOnlyTheFirstQueuedCandidate(t *testing.T) {
	c, texts := advance(t, domain.Capture{Candidates: []domain.CaptureCandidate{candidate("a", "queued"), candidate("b", "queued"), candidate("c", "queued")}})
	if states(c) != "a=pending b=queued c=queued" || c.Candidates[0].ChangeID != "change-a" || c.Candidates[1].ChangeID != "" {
		t.Fatalf("%s", states(c))
	}
	if len(texts) != 2 || texts[0] != "minder: Appointment 1 of 3: A" || !strings.HasPrefix(texts[1], "preview") {
		t.Fatalf("%q", texts)
	}
}

func TestAdvanceDoesNothingWhileOneIsOpen(t *testing.T) {
	for _, open := range []string{"pending", "needs-answer"} {
		c, texts := advance(t, domain.Capture{Candidates: []domain.CaptureCandidate{candidate("a", open), candidate("b", "queued")}})
		if states(c) != "a="+open+" b=queued" || len(texts) != 0 {
			t.Fatalf("%s: %s %q", open, states(c), texts)
		}
	}
}

func TestAdvanceAsksInsteadOfPreviewingWhenThereIsAQuestion(t *testing.T) {
	asking := candidate("a", "queued")
	asking.Question = "How long will it take?"
	c, texts := advance(t, domain.Capture{Candidates: []domain.CaptureCandidate{asking}})
	if states(c) != "a=needs-answer" || c.Candidates[0].ChangeID != "" {
		t.Fatalf("%s", states(c))
	}
	if len(texts) != 1 || !strings.HasPrefix(texts[0], "question: ") {
		t.Fatalf("a question entry was expected, got %q", texts)
	}
}

func TestAPositionCountsOnlyLiveCandidates(t *testing.T) {
	c, texts := advance(t, domain.Capture{Candidates: []domain.CaptureCandidate{candidate("a", "confirmed"), candidate("old", "superseded"), candidate("b", "queued"), candidate("c", "queued")}})
	if states(c) != "a=confirmed old=superseded b=pending c=queued" || texts[0] != "minder: Appointment 2 of 3: B" {
		t.Fatalf("%s %q", states(c), texts)
	}
}

func TestASingleCandidateIsNotAnnounced(t *testing.T) {
	_, texts := advance(t, domain.Capture{Candidates: []domain.CaptureCandidate{candidate("a", "queued")}})
	if len(texts) != 1 || !strings.HasPrefix(texts[0], "preview") {
		t.Fatalf("%q", texts)
	}
}

func TestSummaryCountsAddedAndSkipped(t *testing.T) {
	done := domain.Capture{Candidates: []domain.CaptureCandidate{candidate("a", "confirmed"), candidate("b", "discarded"), candidate("c", "confirmed")}}
	if text, ok := review.Summary(done); !ok || text != "Done: 2 added, 1 skipped." {
		t.Fatalf("%q %v", text, ok)
	}
	for name, c := range map[string]domain.Capture{
		"one open":        {Candidates: []domain.CaptureCandidate{candidate("a", "confirmed"), candidate("b", "pending")}},
		"one queued":      {Candidates: []domain.CaptureCandidate{candidate("a", "confirmed"), candidate("b", "queued")}},
		"a single answer": {Candidates: []domain.CaptureCandidate{candidate("a", "confirmed")}},
	} {
		if text, ok := review.Summary(c); ok {
			t.Fatalf("%s: summary %q", name, text)
		}
	}
}

func TestAnnouncementsSayItemsWhenNotEverythingIsAnAppointment(t *testing.T) {
	task := candidate("t", "queued")
	task.Proposal.Kind = ""
	_, texts := advance(t, domain.Capture{Candidates: []domain.CaptureCandidate{candidate("a", "queued"), task}})
	if texts[0] != "minder: Item 1 of 2: A" {
		t.Fatalf("%q", texts)
	}
}
