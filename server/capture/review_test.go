package capture_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
)

func hotel(question string) domain.CaptureCandidate {
	return domain.CaptureCandidate{Question: question, Evidence: []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}, Proposal: domain.Proposal{Action: "create_item", Title: "Hotel", Kind: "event", StartDate: "2026-11-12", EndDate: "2026-11-15"}}
}

// publish ingests and prepares an email, then commits the agent's review of it.
func (f fixture) publish(t *testing.T, request, notification string, candidates ...domain.CaptureCandidate) string {
	t.Helper()
	id := f.ingest(t, request, plainMail)
	f.prepare(t)
	turn, err := f.db.ClaimAgentTurn(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	patch := &domain.ReviewPatch{SourceID: id, ExpectedRevision: 1, Summary: "One review.", Notification: notification, Candidates: candidates}
	if _, err = command.New(f.db).CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch}); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestAReviewIsPreviewedBeforeSaveAndConfirmedOnceWithTheOriginalKept(t *testing.T) {
	f := newFixture(t)
	id := f.publish(t, "review", "A hotel stay needs review.", hotel(""))
	if items, _ := f.db.Items(); len(items) != 0 {
		t.Fatal("publishing saved an item")
	}
	c, _ := f.svc.Get(id)
	if len(c.Candidates) != 1 || c.Candidates[0].ChangeID == "" || c.Candidates[0].State != "pending" {
		t.Fatalf("%+v", c.Candidates)
	}
	commands := command.New(f.db)
	first, err := commands.Confirm(c.Candidates[0].ChangeID)
	again, err2 := commands.Confirm(c.Candidates[0].ChangeID)
	if err != nil || err2 != nil || first.ID != again.ID {
		t.Fatal(err, err2)
	}
	if c, _ = f.svc.Get(id); c.Candidates[0].State != "confirmed" || c.Status != "reviewed" {
		t.Fatalf("%+v", c)
	}
	items, _ := f.db.Items()
	if len(items) != 1 || len(items[0].SourceIDs) != 1 || items[0].SourceIDs[0] != id {
		t.Fatalf("the item must link to its source: %+v", items)
	}
	reader, _, err := f.svc.Attachment(id, "original")
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()
}

func TestTheSourceStaysReadableAfterConfirmationUntilTheOwnerDeletesIt(t *testing.T) {
	f := newFixture(t)
	id := f.publish(t, "after", "n", hotel(""))
	c, _ := f.svc.Get(id)
	commands := command.New(f.db)
	if _, err := commands.Confirm(c.Candidates[0].ChangeID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ReadSource(context.Background(), id); err != nil {
		t.Fatalf("a confirmed review must keep its source readable: %v", err)
	}
	preview, err := commands.Preview(domain.Proposal{Action: "delete_capture", CaptureID: id, ExpectedVersion: c.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.ReadSource(context.Background(), id); err != nil {
		t.Fatalf("the source went before the deletion was confirmed: %v", err)
	}
	if _, err = commands.Confirm(preview.Change.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.ReadSource(context.Background(), id); !errors.Is(err, domain.ErrSourceUnavailable) {
		t.Fatalf("a deleted source must be unavailable: %v", err)
	}
	if found, _ := f.svc.FindSources(context.Background(), "checkups"); len(found) != 0 {
		t.Fatalf("a deleted source was found: %+v", found)
	}
	if _, _, err = f.svc.Attachment(id, "original"); err == nil {
		t.Fatal("deleted source accessible")
	}
	history, _ := f.db.History(0, 200)
	for _, entry := range history {
		if entry.Capture != nil && (entry.Capture.Sender != "" || len(entry.Capture.Candidates) > 0) {
			t.Fatal("source data remained in history")
		}
	}
}

func TestAPastedSourceIsDeletedWithItsReviewAndItsTextIsRedacted(t *testing.T) {
	f := newFixture(t)
	entry, _ := f.db.AddEntry(domain.Entry{Kind: "user", Text: "Hotel from 12 to 15 November, SECRETREF"})
	_, _, err := f.db.EnqueueAgentTurn(domain.AgentTurn{ID: "t", Kind: domain.TurnOwner}, &domain.Entry{Kind: "user", Text: "Please review this"})
	if err != nil {
		t.Fatal(err)
	}
	turn, _ := f.db.ClaimAgentTurn(time.Now())
	patch := &domain.ReviewPatch{SourceID: entry.ID, Summary: "One review.", Candidates: []domain.CaptureCandidate{func() domain.CaptureCandidate {
		c := hotel("")
		c.Evidence = []domain.CaptureEvidence{{Part: "text", Excerpt: "Hotel from 12 to 15 November"}}
		return c
	}()}}
	commands := command.New(f.db)
	if _, err = commands.CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch}); err != nil {
		t.Fatal(err)
	}
	if doc, err := f.svc.ReadSource(context.Background(), entry.ID); err != nil || doc.Source.Kind != domain.SourceChat {
		t.Fatalf("%+v %v", doc, err)
	}
	c, _ := f.svc.Get(entry.ID)
	preview, err := commands.Preview(domain.Proposal{Action: "delete_capture", CaptureID: entry.ID, ExpectedVersion: c.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = commands.Confirm(preview.Change.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.ReadSource(context.Background(), entry.ID); !errors.Is(err, domain.ErrSourceUnavailable) {
		t.Fatalf("%v", err)
	}
	history, _ := f.db.History(0, 200)
	for _, e := range history {
		if strings.Contains(e.Text, "SECRETREF") {
			t.Fatal("the pasted text outlived its deletion")
		}
	}
}

func TestADeletedSourceCannotBeConfirmedFromAnOldPreview(t *testing.T) {
	f := newFixture(t)
	id := f.publish(t, "stale", "n", hotel(""))
	c, _ := f.svc.Get(id)
	commands := command.New(f.db)
	preview, _ := commands.Preview(domain.Proposal{Action: "delete_capture", CaptureID: id, ExpectedVersion: c.Revision})
	commands.Confirm(preview.Change.ID)
	if _, err := commands.Confirm(c.Candidates[0].ChangeID); err == nil {
		t.Fatal("a deleted source's preview was accepted")
	}
}

func TestDismissClosesAnOpenReviewWithoutChangingItemsAndItsQuestionEntry(t *testing.T) {
	for name, candidates := range map[string][]domain.CaptureCandidate{
		"a question": {hotel("Which room?")}, "a preview": {hotel("")}, "zero items": nil,
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			id := f.publish(t, "dismiss", "n", candidates...)
			c, _ := f.svc.Get(id)
			if _, err := f.svc.Dismiss(context.Background(), id, c.Revision); err != nil {
				t.Fatal(err)
			}
			c, _ = f.svc.Get(id)
			if c.Status != "reviewed" {
				t.Fatalf("%+v", c)
			}
			for _, candidate := range c.Candidates {
				if candidate.State != "discarded" {
					t.Fatalf("%+v", candidate)
				}
			}
			if items, _ := f.db.Items(); len(items) != 0 {
				t.Fatal("dismiss changed items")
			}
			history, _ := f.db.History(0, 100)
			for _, e := range history {
				if e.Kind == "question" {
					t.Fatalf("a question stayed open after dismiss: %+v", e)
				}
			}
			if _, err := f.svc.ReadSource(context.Background(), id); err != nil {
				t.Fatalf("dismiss must keep the source: %v", err)
			}
		})
	}
}

func TestTheFirstPushUsesTheModelNotificationOrTheFallbackAndAnswersPushNothing(t *testing.T) {
	for notification, want := range map[string]string{"A hotel stay needs review.": "A hotel stay needs review.", "": "An email needs your review."} {
		f := newFixture(t)
		f.publish(t, "push", notification, hotel(""))
		deliver(f.svc)
		if len(f.push.bodies) != 1 || f.push.bodies[0] != want {
			t.Fatalf("%q: %q", notification, f.push.bodies)
		}
		deliver(f.svc)
		if len(f.push.bodies) != 1 {
			t.Fatalf("a push repeated: %q", f.push.bodies)
		}
	}
}

func TestAnAnswerRevisionDoesNotPushAndTheStaleFirstPushIsDropped(t *testing.T) {
	f := newFixture(t)
	id := f.publish(t, "answer", "n", hotel("Which room?"))
	c, _ := f.svc.Get(id)
	candidate := c.Candidates[0]
	update := hotel("")
	update.ID, update.Revision = candidate.ID, candidate.Revision
	_, _, err := f.db.EnqueueAgentTurn(domain.AgentTurn{ID: "t", Kind: domain.TurnReviewReply}, &domain.Entry{Kind: "user", Text: "Room 4"})
	if err != nil {
		t.Fatal(err)
	}
	turn, _ := f.db.ClaimAgentTurn(time.Now())
	patch := &domain.ReviewPatch{SourceID: id, ExpectedRevision: c.Revision, Candidates: []domain.CaptureCandidate{update}}
	if _, err = command.New(f.db).CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch}); err != nil {
		t.Fatal(err)
	}
	deliver(f.svc)
	if len(f.push.bodies) != 0 {
		t.Fatalf("the owner's own answer pushed: %q", f.push.bodies)
	}
}

func TestAnAnsweredQuestionEntryBecomesAClosedNote(t *testing.T) {
	f := newFixture(t)
	id := f.publish(t, "close", "n", hotel("Which room?"))
	c, _ := f.svc.Get(id)
	candidate := c.Candidates[0]
	update := hotel("")
	update.ID, update.Revision = candidate.ID, candidate.Revision
	f.db.EnqueueAgentTurn(domain.AgentTurn{ID: "t", Kind: domain.TurnReviewReply}, &domain.Entry{Kind: "user", Text: "Room 4"})
	turn, _ := f.db.ClaimAgentTurn(time.Now())
	patch := &domain.ReviewPatch{SourceID: id, ExpectedRevision: c.Revision, Candidates: []domain.CaptureCandidate{update}}
	if _, err := command.New(f.db).CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch}); err != nil {
		t.Fatal(err)
	}
	history, _ := f.db.History(0, 100)
	closed, open := false, false
	for _, e := range history {
		closed = closed || (e.Kind == "minder" && e.Text == "Questions answered.")
		open = open || e.Kind == "question"
	}
	if !closed || open {
		t.Fatalf("closed=%v open=%v", closed, open)
	}
}

func TestAcknowledgingUnreadablePartsLetsTheReviewFinish(t *testing.T) {
	f := newFixture(t)
	id := f.ingest(t, "ack", "From: a@example.com\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=t.bin\r\n\r\nBinary")
	f.prepare(t)
	turn, _ := f.db.ClaimAgentTurn(time.Now())
	patch := &domain.ReviewPatch{SourceID: id, ExpectedRevision: 1, Candidates: []domain.CaptureCandidate{hotel("")}}
	if _, err := command.New(f.db).CommitAgentTurn(turn, domain.AgentOutcome{TurnID: turn.ID, Review: patch}); err != nil {
		t.Fatal(err)
	}
	c, _ := f.svc.Get(id)
	if c.Status != "needs-answer" {
		t.Fatalf("unreadable parts must keep the review open: %+v", c)
	}
	if _, err := f.svc.Acknowledge(context.Background(), id, c.Revision); err != nil {
		t.Fatal(err)
	}
	if c, _ = f.svc.Get(id); !c.CoverageAcknowledged || c.Status == "needs-answer" {
		t.Fatalf("%+v", c)
	}
}
