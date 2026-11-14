package capture_test

import (
	"context"
	"testing"
	"time"
)

func TestADurationAnswerSkipsTheModel(t *testing.T) {
	x := &threeAppointments{}
	svc, _, commands, id := flowFixture(t, x)
	c, _ := svc.Get(id)
	if _, e := svc.AnswerCandidate(context.Background(), id, c.Revision, "a", "1 hour"); e != nil {
		t.Fatal(e)
	}
	c, _ = svc.Get(id)
	if x.calls != 1 || c.Status == "received" {
		t.Fatalf("the model ran again: calls %d, status %s", x.calls, c.Status)
	}
	a := c.Candidates[0]
	if a.State != "pending" || a.ChangeID == "" || a.Question != "" || a.Proposal.DurationMinutes != 60 {
		t.Fatalf("%+v", a)
	}
	if _, e := commands.Confirm(a.ChangeID); e != nil {
		t.Fatalf("the preview cannot be confirmed: %v", e)
	}
	if got := stateOf(t, svc, id); got != "a=confirmed b=needs-answer c=queued" {
		t.Fatalf("states %s", got)
	}
	c, _ = svc.Get(id)
	if _, e := svc.AnswerCandidate(context.Background(), id, c.Revision, "b", "30 minutes"); e != nil {
		t.Fatal(e)
	}
	if got := stateOf(t, svc, id); got != "a=confirmed b=pending c=queued" || x.calls != 1 {
		t.Fatalf("states %s, model calls %d", got, x.calls)
	}
}

func TestAnAnswerThatIsNotADurationGoesToTheModel(t *testing.T) {
	x := &threeAppointments{}
	svc, _, _, id := flowFixture(t, x)
	c, _ := svc.Get(id)
	if _, e := svc.AnswerCandidate(context.Background(), id, c.Revision, "a", "about an hour"); e != nil {
		t.Fatal(e)
	}
	if c, _ = svc.Get(id); c.Status != "received" {
		t.Fatalf("status %s", c.Status)
	}
	if e := svc.ProcessNext(context.Background(), time.Now()); e != nil || x.calls != 2 {
		t.Fatalf("%v calls %d", e, x.calls)
	}
}

func TestADurationOnADifferentQuestionGoesToTheModel(t *testing.T) {
	svc, _, id := processed(t, questionsExtractor{})
	c, _ := svc.Get(id)
	if _, e := svc.AnswerCandidate(context.Background(), id, c.Revision, "question", "1 hour"); e != nil {
		t.Fatal(e)
	}
	if c, _ = svc.Get(id); c.Status != "received" {
		t.Fatalf("a duration answered a question about a date: status %s", c.Status)
	}
}
