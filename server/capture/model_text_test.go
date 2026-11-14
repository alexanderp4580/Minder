package capture_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
)

type modelText struct {
	summary, notification string
	dropped               int
}

func (x modelText) Extract(ctx context.Context, doc domain.CaptureDocument, answers []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	result, err := singleEvent{}.Extract(ctx, doc, answers)
	result.Summary, result.Notification, result.Dropped = x.summary, x.notification, x.dropped
	return result, err
}

func TestModelTextIsPostedAndPersistedUnchanged(t *testing.T) {
	svc, db, _, id := flowFixture(t, modelText{"  Three visits, starting with the dentist.  ", "A dental visit needs a decision.", 2})
	texts := chatTexts(t, db)
	if len(texts) != 2 || texts[0] != "  Three visits, starting with the dentist.  " || texts[1] != "I could not verify 2 appointments from this email." {
		t.Fatalf("%q", texts)
	}
	c, err := svc.Get(id)
	if err != nil || c.Notification != "A dental visit needs a decision." {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestMissingSummaryPostsNoOpening(t *testing.T) {
	_, db, _, _ := flowFixture(t, modelText{})
	if texts := chatTexts(t, db); len(texts) != 0 {
		t.Fatalf("unexpected opening %q", texts)
	}
}

func TestFirstPushUsesModelTextOrFallback(t *testing.T) {
	for _, text := range []string{"Review the three dental visits.", ""} {
		svc, sender, _ := notified(t, modelText{notification: text})
		if err := svc.ProcessNext(context.Background(), time.Now()); err != nil {
			t.Fatal(err)
		}
		deliver(svc)
		want := text
		if want == "" {
			want = "An email needs your review."
		}
		if len(sender.bodies) != 1 || sender.bodies[0] != want {
			t.Fatalf("%q", sender.bodies)
		}
	}
}

func TestAnswersDoNotRepeatTheModelOpening(t *testing.T) {
	svc, db, _, id := flowFixture(t, &threeAppointments{})
	c, _ := svc.Get(id)
	if _, err := svc.AnswerCandidate(context.Background(), id, c.Revision, "a", "1 hour"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ProcessNext(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(strings.Join(chatTexts(t, db), "|"), "Three visits await your review."); got != 1 {
		t.Fatalf("opening count %d", got)
	}
}
