package capture_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/capture"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/mailparse"
	"github.com/alexanderp4580/Minder/server/store"
	"path/filepath"
)

// pushes records the body of every push sent.
type pushes struct{ bodies []string }

func (p *pushes) Send(_ context.Context, _ string, data map[string]string) (bool, error) {
	p.bodies = append(p.bodies, data["body"])
	return false, nil
}

// notified builds a capture service with one registered phone and a recording sender.
func notified(t *testing.T, x capture.Extractor) (*capture.Service, *pushes, string) {
	t.Helper()
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if e = db.RegisterDevice("phone", "token"); e != nil {
		t.Fatal(e)
	}
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, x)
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	sender := &pushes{}
	svc.Sender = sender
	id, _, e := svc.Ingest(context.Background(), domain.CaptureEnvelope{From: "owner@example.com", RequestID: "push"}, []byte("Subject: Checkups\r\nFrom: owner@example.com\r\n\r\nHotel"))
	if e != nil {
		t.Fatal(e)
	}
	return svc, sender, id
}

// deliver runs one worker pass that sends the queued pushes and then stops.
func deliver(svc *capture.Service) {
	stopped, cancel := context.WithCancel(context.Background())
	cancel()
	svc.Run(stopped)
}

func TestTheFirstPushCountsTheAppointments(t *testing.T) {
	svc, sender, _ := notified(t, &threeAppointments{})
	if e := svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	deliver(svc)
	if len(sender.bodies) != 1 || sender.bodies[0] != "3 appointments need your review" {
		t.Fatalf("%q", sender.bodies)
	}
}

func TestOneAppointmentReadsInTheSingular(t *testing.T) {
	svc, sender, _ := notified(t, singleEvent{})
	if e := svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	deliver(svc)
	if len(sender.bodies) != 1 || sender.bodies[0] != "1 appointment needs your review" {
		t.Fatalf("%q", sender.bodies)
	}
}

type singleEvent struct{}

func (singleEvent) Extract(context.Context, domain.CaptureDocument, []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	return domain.CaptureExtraction{Candidates: []domain.CaptureCandidate{{ID: "a", Evidence: []domain.CaptureEvidence{{Part: "1", Excerpt: "Hotel"}}, Proposal: domain.Proposal{Action: "create_item", Title: "Dentist", Kind: "event", StartAt: "2026-11-20T10:00:00+01:00", DurationMinutes: 30}}}}, nil
}

func TestAnAnswerQueuesNoPush(t *testing.T) {
	svc, sender, id := notified(t, &threeAppointments{})
	svc.ProcessNext(context.Background(), time.Now())
	deliver(svc)
	c, _ := svc.Get(id)
	if _, e := svc.AnswerCandidate(context.Background(), id, c.Revision, "a", "1 hour"); e != nil {
		t.Fatal(e)
	}
	svc.ProcessNext(context.Background(), time.Now())
	deliver(svc)
	if len(sender.bodies) != 1 {
		t.Fatalf("the owner's own answer sent a push: %q", sender.bodies)
	}
}

func TestAFailurePushesOnce(t *testing.T) {
	svc, sender, _ := notified(t, &flaky{err: &domain.ModelError{Message: "x", Reason: domain.ReasonCutOff}, fails: 100})
	if e := svc.ProcessNext(context.Background(), time.Now()); e == nil {
		t.Fatal("expected the failure")
	}
	deliver(svc)
	deliver(svc)
	if len(sender.bodies) != 1 || sender.bodies[0] != "Minder could not read an email." {
		t.Fatalf("%q", sender.bodies)
	}
}

func TestARetryThatWorksPushesTheReviewAfterTheFailurePush(t *testing.T) {
	svc, sender, id := notified(t, &flaky{err: errors.New("down"), fails: 1})
	svc.ProcessNext(context.Background(), time.Now())
	deliver(svc)
	if _, e := svc.Retry(id); e != nil {
		t.Fatal(e)
	}
	svc.ProcessNext(context.Background(), time.Now())
	deliver(svc)
	if len(sender.bodies) != 2 || sender.bodies[1] != "1 appointment needs your review" {
		t.Fatalf("%q", sender.bodies)
	}
}
