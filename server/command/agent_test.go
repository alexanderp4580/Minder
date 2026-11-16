package command_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/port"
	"github.com/alexanderp4580/Minder/server/store"
)

type agentFixture struct {
	db       *store.Store
	commands *command.Service
	source   string
}

// newAgent prepares an email the way preparation leaves it: a processing capture with its
// card, source reference and queued arrival turn.
func newAgent(t *testing.T) agentFixture {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = db.RegisterDevice("phone", "token"); err != nil {
		t.Fatal(err)
	}
	const id = "mail"
	err = db.Transact(func(tx port.Transaction) error {
		c := domain.Capture{ID: id, Status: "processing", Revision: 1, Prepared: true, Subject: "Checkups", EntryID: "card", Candidates: []domain.CaptureCandidate{}, Coverage: []domain.CaptureCoverage{}, Attachments: []domain.CaptureAttachment{}}
		if _, err := tx.PutEntry(domain.Entry{ID: "card", Kind: "capture", Text: "Reading the email.", Capture: &c}); err != nil {
			return err
		}
		if err := tx.PutSource(domain.SourceRef{ID: id, Kind: domain.SourceEmail, CaptureID: id, EntryID: "card", Label: "Checkups"}); err != nil {
			return err
		}
		return tx.PutCapture(c)
	})
	if err != nil {
		t.Fatal(err)
	}
	return agentFixture{db, command.New(db), id}
}

// arrive queues the email's arrival turn and claims it, as the worker would.
func (f agentFixture) arrive(t *testing.T) domain.AgentTurn {
	t.Helper()
	if err := f.db.Transact(func(tx port.Transaction) error { return tx.PutAgentTurn(domain.ArrivalTurn(f.source, 1)) }); err != nil {
		t.Fatal(err)
	}
	return f.claim(t)
}

func (f agentFixture) claim(t *testing.T) domain.AgentTurn {
	t.Helper()
	turn, err := f.db.ClaimAgentTurn(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return turn
}

func (f agentFixture) owner(t *testing.T, text string) domain.AgentTurn {
	t.Helper()
	if _, _, err := f.db.EnqueueAgentTurn(domain.AgentTurn{ID: domain.NewID(), Kind: domain.TurnOwner}, &domain.Entry{Kind: "user", Text: text}); err != nil {
		t.Fatal(err)
	}
	return f.claim(t)
}

// candidate builds a proposal. An empty id makes it a new candidate; the server assigns its ID.
func candidate(id, title, question string) domain.CaptureCandidate {
	return domain.CaptureCandidate{ID: id, Question: question, Evidence: []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}, Proposal: domain.Proposal{Action: "create_item", Title: title, Kind: "event", StartAt: "2026-11-20T10:00:00+01:00"}}
}

// fourItems is the synthetic review the plan calls for: three appointments and a questionnaire.
func fourItems() *domain.ReviewPatch {
	task := domain.CaptureCandidate{Evidence: []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}, Proposal: domain.Proposal{Action: "create_item", Title: "Fill in the questionnaire", DueDate: "2026-11-18"}}
	return &domain.ReviewPatch{SourceID: "mail", ExpectedRevision: 1, Summary: "Three visits and a form await your review.", Notification: "Three dental visits need a decision.",
		Candidates: []domain.CaptureCandidate{candidate("", "Dentist", "How long will it take?"), candidate("", "Checkup", "How long will it take?"), candidate("", "Follow-up", ""), task}}
}

func (f agentFixture) publish(t *testing.T) domain.AgentTurn {
	t.Helper()
	turn := f.arrive(t)
	if _, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: fourItems(), CostEUR: 0.01}); err != nil {
		t.Fatal(err)
	}
	return turn
}

func (f agentFixture) capture(t *testing.T) domain.Capture {
	t.Helper()
	c, err := f.db.Capture(f.source)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// shortName names a candidate in a state listing by its title's first word, lower case.
func shortName(title string) string {
	return strings.ToLower(strings.Fields(title)[0])
}

// idOf finds a candidate's server-assigned ID by its title.
func idOf(c domain.Capture, title string) string {
	for _, candidate := range c.Candidates {
		if candidate.Proposal.Title == title {
			return candidate.ID
		}
	}
	return ""
}

func states(c domain.Capture) string {
	var out []string
	for _, candidate := range c.Candidates {
		out = append(out, candidate.Proposal.CandidateID[:0]+shortName(candidate.Proposal.Title)+"="+candidate.State)
	}
	return strings.Join(out, " ")
}

func (f agentFixture) texts(t *testing.T) []string {
	t.Helper()
	history, _ := f.db.History(0, 200)
	out := []string{}
	for _, entry := range history {
		if entry.Kind == "minder" || entry.Kind == "question" || entry.Kind == "error" {
			out = append(out, entry.Kind+": "+entry.Text)
		}
	}
	return out
}

func TestTheFirstReviewPublishesOnlyTheFirstCandidateAndKeepsTheModelText(t *testing.T) {
	f := newAgent(t)
	f.publish(t)
	c := f.capture(t)
	if states(c) != "dentist=needs-answer checkup=queued follow-up=queued fill=queued" || c.Notification != "Three dental visits need a decision." || c.Revision != 1 || c.Status != "needs-answer" {
		t.Fatalf("%s %+v", states(c), c)
	}
	texts := f.texts(t)
	if len(texts) < 3 || texts[0] != "minder: Three visits and a form await your review." || texts[1] != "minder: Item 1 of 4: Dentist" || !strings.HasPrefix(texts[2], "question: ") {
		t.Fatalf("%q", texts)
	}
	if c.Candidates[1].ChangeID != "" || c.Candidates[0].Proposal.CandidateID != c.Candidates[0].ID || c.Candidates[0].Proposal.CaptureID != "mail" {
		t.Fatalf("%s", states(c))
	}
}

func TestTheFirstReviewQueuesOnePushWithTheModelNotification(t *testing.T) {
	f := newAgent(t)
	f.publish(t)
	deliveries, err := f.db.CaptureDeliveries(time.Now())
	if err != nil || len(deliveries) != 1 || deliveries[0].CaptureID != "mail" || deliveries[0].Revision != 1 {
		t.Fatalf("%+v %v", deliveries, err)
	}
}

func TestTheTurnEndsDoneWithItsEntriesAndTheCostOnTheFirst(t *testing.T) {
	f := newAgent(t)
	turn := f.arrive(t)
	entries, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: fourItems(), CostEUR: 0.01})
	if err != nil || len(entries) == 0 || entries[0].CostEUR != 0.01 {
		t.Fatalf("%+v %v", entries, err)
	}
	for _, e := range entries[1:] {
		if e.CostEUR != 0 {
			t.Fatalf("cost repeated on %+v", e)
		}
	}
	stored := f.turn(t, turn.ID)
	if stored.State != domain.TurnDone || len(stored.EntryIDs) != len(entries) {
		t.Fatalf("%+v", stored)
	}
}

func (f agentFixture) turn(t *testing.T, id string) (turn domain.AgentTurn) {
	t.Helper()
	err := f.db.Transact(func(tx port.Transaction) (e error) { turn, e = tx.AgentTurn(id); return })
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestCommittingATurnTwiceReturnsItsEntriesAndWritesNothingMore(t *testing.T) {
	f := newAgent(t)
	turn := f.arrive(t)
	outcome := domain.AgentOutcome{TurnID: turn.ID, Review: fourItems()}
	first, err := f.commands.CommitAgentTurn(turn, outcome)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := f.db.History(0, 200)
	again, err := f.commands.CommitAgentTurn(turn, outcome)
	after, _ := f.db.History(0, 200)
	if err != nil || len(again) != len(first) || again[0].ID != first[0].ID || len(before) != len(after) {
		t.Fatalf("%v %d %d", err, len(before), len(after))
	}
	if d, _ := f.db.CaptureDeliveries(time.Now()); len(d) != 1 {
		t.Fatalf("pushes %d", len(d))
	}
}

func TestATargetedUpdateChangesOnlyItsCandidateAndKeepsTheRest(t *testing.T) {
	f := newAgent(t)
	f.publish(t)
	c := f.capture(t)
	answer := f.owner(t, "1 hour")
	update := candidate(idOf(c, "Dentist"), "Dentist", "")
	update.Revision = c.Candidates[0].Revision
	update.Evidence = nil // an answer needs no new quote; the evidence stays
	update.Proposal.DurationMinutes = 60
	outcome := domain.AgentOutcome{TurnID: answer.ID, Review: &domain.ReviewPatch{SourceID: "mail", ExpectedRevision: c.Revision, Candidates: []domain.CaptureCandidate{update}}}
	if _, err := f.commands.CommitAgentTurn(answer, outcome); err != nil {
		t.Fatal(err)
	}
	after := f.capture(t)
	if states(after) != "dentist=pending checkup=queued follow-up=queued fill=queued" || after.Revision != 2 {
		t.Fatalf("%s revision %d", states(after), after.Revision)
	}
	if got := after.Candidates[0]; got.Proposal.DurationMinutes != 60 || len(got.Evidence) != 1 || got.Revision != 2 || got.Proposal.CaptureRevision != 2 || got.ChangeID == "" {
		t.Fatalf("%+v", got)
	}
	if after.Candidates[1].Question != "How long will it take?" || after.Candidates[1].Revision != 1 {
		t.Fatalf("another candidate changed: %+v", after.Candidates[1])
	}
	if d, _ := f.db.CaptureDeliveries(time.Now()); len(d) != 0 {
		t.Fatalf("an answer queued a push for its revision: %d", len(d))
	}
}

func TestAnUpdateKeepsAConfirmedItemAndTheOthersWhenAnotherCandidateChanges(t *testing.T) {
	f := newAgent(t)
	f.publish(t)
	reply := func(title, text string, minutes int) {
		c := f.capture(t)
		var current domain.CaptureCandidate
		for _, candidate := range c.Candidates {
			if candidate.Proposal.Title == title {
				current = candidate
			}
		}
		turn := f.owner(t, text)
		update := candidate(current.ID, title, "")
		update.Revision, update.Evidence = current.Revision, nil
		update.Proposal.DurationMinutes = minutes
		if _, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: &domain.ReviewPatch{SourceID: "mail", ExpectedRevision: c.Revision, Candidates: []domain.CaptureCandidate{update}}}); err != nil {
			t.Fatal(err)
		}
	}
	reply("Dentist", "1 hour", 60)
	if _, err := f.commands.Confirm(f.capture(t).Candidates[0].ChangeID); err != nil {
		t.Fatal(err)
	}
	if got := states(f.capture(t)); got != "dentist=confirmed checkup=needs-answer follow-up=queued fill=queued" {
		t.Fatal(got)
	}
	reply("Checkup", "30 minutes", 30)
	final := f.capture(t)
	if states(final) != "dentist=confirmed checkup=pending follow-up=queued fill=queued" || len(final.Candidates) != 4 || final.Candidates[0].ItemID == "" {
		t.Fatalf("%s", states(final))
	}
}

func TestAStaleReviewRevisionOrCandidateRevisionIsRefusedWithoutPartialChanges(t *testing.T) {
	f := newAgent(t)
	f.publish(t)
	c := f.capture(t)
	before, _ := f.db.History(0, 200)
	good := candidate(idOf(c, "Dentist"), "Dentist", "")
	good.Revision = c.Candidates[0].Revision
	for name, patch := range map[string]*domain.ReviewPatch{
		"review revision": {SourceID: "mail", ExpectedRevision: c.Revision + 3, Candidates: []domain.CaptureCandidate{good}},
		"candidate revision": {SourceID: "mail", ExpectedRevision: c.Revision, Candidates: []domain.CaptureCandidate{good, func() domain.CaptureCandidate {
			x := candidate(idOf(c, "Checkup"), "Checkup", "")
			x.Revision = 9
			return x
		}()}},
		"unknown candidate": {SourceID: "mail", ExpectedRevision: c.Revision, Candidates: []domain.CaptureCandidate{good, candidate("zzz", "Nothing", "")}},
	} {
		turn := f.owner(t, name)
		_, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch})
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("%s: %v", name, err)
		}
		after := f.capture(t)
		if states(after) != states(c) || after.Revision != c.Revision {
			t.Fatalf("%s changed the review: %s", name, states(after))
		}
		f.failTurn(t, turn, err)
	}
	history, _ := f.db.History(0, 200)
	if len(history) != len(before)+3*2 { // each owner message and its failure entry only
		t.Fatalf("partial writes: %d entries, was %d", len(history), len(before))
	}
}

func (f agentFixture) failTurn(t *testing.T, turn domain.AgentTurn, cause error) {
	t.Helper()
	if err := f.commands.FailAgentTurn(turn, cause); err != nil {
		t.Fatal(err)
	}
}

func TestAConfirmedOrDiscardedCandidateCannotBeWritten(t *testing.T) {
	f := newAgent(t)
	f.publish(t)
	turn := f.owner(t, "answer")
	c := f.capture(t)
	update := candidate(idOf(c, "Dentist"), "Dentist", "")
	update.Revision, update.Proposal.DurationMinutes = c.Candidates[0].Revision, 60
	f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: &domain.ReviewPatch{SourceID: "mail", ExpectedRevision: c.Revision, Candidates: []domain.CaptureCandidate{update}}})
	c = f.capture(t)
	if _, err := f.commands.Discard(c.Candidates[0].ChangeID); err != nil {
		t.Fatal(err)
	}
	c = f.capture(t)
	turn = f.owner(t, "again")
	update.Revision = c.Candidates[0].Revision
	_, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: &domain.ReviewPatch{SourceID: "mail", ExpectedRevision: c.Revision, Candidates: []domain.CaptureCandidate{update}}})
	if !errors.Is(err, domain.ErrConflict) || f.capture(t).Candidates[0].State != "discarded" {
		t.Fatalf("%v %s", err, states(f.capture(t)))
	}
}

func TestAnAnswerInTextLeavesTheOpenQuestionAndEveryCandidateAlone(t *testing.T) {
	f := newAgent(t)
	f.publish(t)
	before := f.capture(t)
	turn := f.owner(t, "Where is the dentist?")
	entries, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Text: "Main Street 1.", CostEUR: 0.002})
	if err != nil || len(entries) != 1 || entries[0].Kind != "minder" || entries[0].CostEUR != 0.002 {
		t.Fatalf("%+v %v", entries, err)
	}
	after := f.capture(t)
	if states(after) != states(before) || after.Revision != before.Revision || after.Candidates[0].Question != before.Candidates[0].Question {
		t.Fatalf("%s", states(after))
	}
	history, _ := f.db.History(0, 200)
	for _, entry := range history {
		if entry.Kind == "minder" && strings.Contains(entry.Text, "Questions answered") {
			t.Fatal("the open question was closed by an unrelated answer")
		}
	}
}

func TestAnArrivalWithNothingActionableClosesTheReviewAndSaysSo(t *testing.T) {
	f := newAgent(t)
	turn := f.arrive(t)
	if _, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Text: "This is a newsletter. Nothing to do."}); err != nil {
		t.Fatal(err)
	}
	c := f.capture(t)
	if c.Status != "reviewed" || len(c.Candidates) != 0 {
		t.Fatalf("%+v", c)
	}
	if texts := f.texts(t); len(texts) != 1 || texts[0] != "minder: This is a newsletter. Nothing to do." {
		t.Fatalf("%q", texts)
	}
}

func TestProposalsPreviewTogetherAndAnInvalidOneRollsBackTheWholeTurn(t *testing.T) {
	f := newAgent(t)
	turn := f.owner(t, "Add two things")
	good := domain.Proposal{Action: "create_item", Title: "Call plumber"}
	bad := domain.Proposal{Action: "update_item", ItemID: "missing", ExpectedVersion: 1}
	before, _ := f.db.History(0, 200)
	_, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Proposals: []domain.Proposal{good, bad}})
	if err == nil {
		t.Fatal("an invalid proposal was accepted")
	}
	after, _ := f.db.History(0, 200)
	if len(after) != len(before) || f.turn(t, turn.ID).State == domain.TurnDone {
		t.Fatalf("partial commit: %d entries, turn %+v", len(after), f.turn(t, turn.ID))
	}
	entries, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Proposals: []domain.Proposal{good, {Action: "create_item", Title: "Buy tape"}}, CostEUR: 0.004})
	if err != nil || len(entries) != 2 || entries[0].Kind != "preview" || entries[0].CostEUR != 0.004 || entries[1].CostEUR != 0 {
		t.Fatalf("%+v %v", entries, err)
	}
}

func TestAnOfferIsStoredAsTheFixedFormatEntry(t *testing.T) {
	f := newAgent(t)
	turn := f.owner(t, "Where is Cafe Luna?")
	offer := domain.Offer{Kind: domain.LookupPlaces, Query: "Cafe Luna Utrecht"}
	entries, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Offer: &offer, CostEUR: 0.001})
	want, _ := domain.OfferText(offer)
	if err != nil || len(entries) != 1 || entries[0].Text != want || entries[0].CostEUR != 0.001 {
		t.Fatalf("%+v %v", entries, err)
	}
}

func TestAReviewOfPastedTextCreatesItsSourceAndReviewOnFirstPublication(t *testing.T) {
	f := newAgent(t)
	text := "Dentist on 20 November at 10:00. Bring your card."
	turn := f.owner(t, text)
	var entryID string
	history, _ := f.db.History(0, 20)
	for _, e := range history {
		if e.Kind == "user" {
			entryID = e.ID
		}
	}
	patch := &domain.ReviewPatch{SourceID: entryID, ExpectedRevision: 0, Summary: "One appointment to review.", Notification: "A pasted appointment needs review.",
		Candidates: []domain.CaptureCandidate{{Evidence: []domain.CaptureEvidence{{Part: "text", Excerpt: "Dentist on 20 November at 10:00"}}, Proposal: domain.Proposal{Action: "create_item", Title: "Dentist", Kind: "event", StartAt: "2026-11-20T10:00:00+01:00", DurationMinutes: 30}}}}
	if _, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch}); err != nil {
		t.Fatal(err)
	}
	c, err := f.db.Capture(entryID)
	if err != nil || c.Kind != "chat" || c.Revision != 1 || len(c.Candidates) != 1 || c.Candidates[0].State != "pending" || c.Candidates[0].ID == "" || c.Subject != "Dentist on 20 November at 10:00. Bring your card." {
		t.Fatalf("%+v %v", c, err)
	}
	var ref domain.SourceRef
	f.db.Transact(func(tx port.Transaction) (e error) { ref, e = tx.Source(entryID); return })
	if ref.Kind != domain.SourceChat || ref.EntryID != entryID || ref.CaptureID != entryID || ref.Deleted {
		t.Fatalf("%+v", ref)
	}
	if d, _ := f.db.CaptureDeliveries(time.Now()); len(d) != 1 || d[0].CaptureID != entryID {
		t.Fatalf("the pasted review queues one push for its own source: %+v", d)
	}
}

func TestAReviewOfADeletedOrMissingSourceIsRefused(t *testing.T) {
	f := newAgent(t)
	f.publish(t)
	f.db.Transact(func(tx port.Transaction) error { return tx.RedactCapture("mail") })
	turn := f.owner(t, "More")
	patch := &domain.ReviewPatch{SourceID: "mail", ExpectedRevision: 1, Candidates: []domain.CaptureCandidate{candidate("", "X", "")}}
	_, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("%v", err)
	}
	f.failTurn(t, turn, err)
	turn = f.owner(t, "More")
	patch.SourceID = "nothing"
	if _, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("%v", err)
	}
}

func TestFailingAnOwnerTurnPostsOneVisibleErrorAndEndsTheTurn(t *testing.T) {
	f := newAgent(t)
	turn := f.owner(t, "Hello")
	cause := errors.New("provider exploded")
	f.failTurn(t, turn, cause)
	f.failTurn(t, turn, cause)
	history, _ := f.db.History(0, 50)
	errorsSeen := 0
	for _, e := range history {
		if e.Kind == "error" {
			errorsSeen++
			if e.ID != turn.ID || e.Text != "Minder could not finish. Try again." {
				t.Fatalf("%+v", e)
			}
		}
	}
	if errorsSeen != 1 || f.turn(t, turn.ID).State != domain.TurnFailed {
		t.Fatalf("%d errors, %+v", errorsSeen, f.turn(t, turn.ID))
	}
}

func TestFailingAnArrivalFailsTheSourceWithOneReportAndOnePush(t *testing.T) {
	f := newAgent(t)
	turn := f.arrive(t)
	cause := &domain.ModelError{Message: "x", Reason: domain.ReasonCutOff}
	f.failTurn(t, turn, cause)
	f.failTurn(t, turn, cause)
	c := f.capture(t)
	if c.Status != "failed" || c.Error != domain.ReasonCutOff.Sentence() {
		t.Fatalf("%+v", c)
	}
	history, _ := f.db.History(0, 50)
	reports := 0
	for _, e := range history {
		if e.Kind == "minder" && e.CaptureID == "mail" {
			reports++
		}
	}
	d, _ := f.db.CaptureDeliveries(time.Now())
	if reports != 1 || len(d) != 1 {
		t.Fatalf("%d reports, %d pushes", reports, len(d))
	}
}

func TestFailingAnArrivalForADeletedSourceWritesNothingVisible(t *testing.T) {
	f := newAgent(t)
	turn := f.arrive(t)
	f.db.Transact(func(tx port.Transaction) error {
		c, _ := tx.Capture("mail")
		return tx.PutCapture(domain.Capture{ID: "mail", Deleted: true, Revision: c.Revision + 1, EntryID: c.EntryID})
	})
	before, _ := f.db.History(0, 50)
	f.failTurn(t, turn, errors.New("boom"))
	after, _ := f.db.History(0, 50)
	if len(after) != len(before) || f.turn(t, turn.ID).State != domain.TurnFailed {
		t.Fatalf("%d %d", len(before), len(after))
	}
}

func TestAQuestionAboutPastedTextDoesNotCallItAnEmail(t *testing.T) {
	f := newAgent(t)
	entry, _ := f.db.AddEntry(domain.Entry{Kind: "user", Text: "Dentist on 20 November at 10:00."})
	f.db.EnqueueAgentTurn(domain.AgentTurn{ID: "t", Kind: domain.TurnOwner}, &domain.Entry{Kind: "user", Text: "Review this"})
	turn := f.claim(t)
	ask := candidate("", "Dentist", "How long will it take?")
	ask.Evidence = []domain.CaptureEvidence{{Part: "text", Excerpt: "Dentist"}}
	patch := &domain.ReviewPatch{SourceID: entry.ID, Candidates: []domain.CaptureCandidate{ask}}
	if _, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch}); err != nil {
		t.Fatal(err)
	}
	if texts := f.texts(t); texts[len(texts)-1] != "question: Questions about the pasted text." {
		t.Fatalf("%q", texts)
	}
}

func TestAnAgentProposalUsesTheCurrentItemVersionWhateverTheModelSent(t *testing.T) {
	f := newAgent(t)
	item, err := f.commands.AddTask("Call plumber")
	if err != nil {
		t.Fatal(err)
	}
	if item, err = f.commands.MoveTask(item.ID, item.Version, "in_progress"); err != nil {
		t.Fatal(err)
	}
	for _, sent := range []int{0, 1, 3, 99} {
		turn := f.owner(t, "put the plumber task back")
		entries, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Proposals: []domain.Proposal{{Action: "update_item", ItemID: item.ID, ExpectedVersion: sent, Status: "open"}}})
		if err != nil {
			t.Fatalf("version %d: %v", sent, err)
		}
		if len(entries) != 1 || entries[0].Change == nil || entries[0].Change.Action != "update_item" {
			t.Fatalf("version %d: %+v", sent, entries)
		}
		if _, err = f.commands.Discard(entries[0].Change.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnAgentProposalForAnUnknownItemStillFailsWithNotFound(t *testing.T) {
	f := newAgent(t)
	turn := f.owner(t, "start the plumber task")
	_, err := f.commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Proposals: []domain.Proposal{{Action: "update_item", ItemID: "not-an-item", ExpectedVersion: 1, Status: "in_progress"}}})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
}
