package capture_test

import (
	"context"
	"testing"
	"time"
)

func TestEveryAnswerGoesToTheModel(t *testing.T) {
	for _, answer := range []string{"1 hour", "about an hour", "Skip"} {
		x := &threeAppointments{}
		svc, _, _, id := flowFixture(t, x)
		c, _ := svc.Get(id)
		if _, e := svc.AnswerCandidate(context.Background(), id, c.Revision, "a", answer); e != nil {
			t.Fatal(e)
		}
		if c, _ = svc.Get(id); c.Status != "received" || x.calls != 1 {
			t.Fatalf("%q was handled by code: status %s, model calls %d", answer, c.Status, x.calls)
		}
		if e := svc.ProcessNext(context.Background(), time.Now()); e != nil || x.calls != 2 {
			t.Fatalf("%q: %v, model calls %d", answer, e, x.calls)
		}
	}
}
