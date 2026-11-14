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

// flaky fails the first extractions with err, then succeeds like extracted.
type flaky struct {
	err   error
	fails int
	calls int
}

func (f *flaky) Extract(ctx context.Context, d domain.CaptureDocument, a []domain.CaptureAnswer) (domain.CaptureExtraction, error) {
	f.calls++
	if f.calls <= f.fails {
		return domain.CaptureExtraction{}, f.err
	}
	return extracted{}.Extract(ctx, d, a)
}

func failingService(t *testing.T, x capture.Extractor) (*capture.Service, *store.Store, string) {
	t.Helper()
	db, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	files, _ := mailfiles.New(t.TempDir())
	svc := capture.New(db, files, x)
	svc.Parser = &mailparse.Parser{}
	svc.Preview = command.PreviewIn
	id, _, e := svc.Ingest(context.Background(), domain.CaptureEnvelope{From: "owner@example.com", RequestID: "fail"}, []byte("From: owner@example.com\r\n\r\nHotel"))
	if e != nil {
		t.Fatal(e)
	}
	return svc, db, id
}

func captureEntryText(t *testing.T, db *store.Store) string {
	t.Helper()
	history, _ := db.History(0, 50)
	for n := len(history) - 1; n >= 0; n-- {
		if history[n].Kind == "capture" {
			return history[n].Text
		}
	}
	t.Fatal("no capture entry")
	return ""
}

func TestAFailedExtractionIsNotRetriedAutomatically(t *testing.T) {
	x := &flaky{err: &domain.ModelError{Message: "x", Reason: domain.ReasonEvidence}, fails: 100}
	svc, db, id := failingService(t, x)
	now := time.Now()
	if e := svc.ProcessNext(context.Background(), now); !errors.Is(e, x.err) {
		t.Fatalf("the worker must see the cause: %v", e)
	}
	for _, later := range []time.Duration{time.Second, time.Minute, 5 * time.Minute, time.Hour} {
		if e := svc.ProcessNext(context.Background(), now.Add(later)); e != nil {
			t.Fatalf("a failed email was claimed again: %v", e)
		}
	}
	c, _ := svc.Get(id)
	if x.calls != 1 || c.Status != "failed" || c.NextAt != 0 {
		t.Fatalf("calls %d status %s next %d", x.calls, c.Status, c.NextAt)
	}
	if got := captureEntryText(t, db); got != domain.ReasonEvidence.Sentence() {
		t.Fatalf("entry text %q", got)
	}
}

func TestTheFailureEntryNamesTheReason(t *testing.T) {
	for name, cause := range map[string]error{
		"cut off":  &domain.ModelError{Message: "x", Reason: domain.ReasonCutOff},
		"evidence": &domain.ModelError{Message: "x", Reason: domain.ReasonEvidence},
		"limit":    domain.ErrLimit,
		"other":    errors.New("disk"),
	} {
		t.Run(name, func(t *testing.T) {
			svc, db, _ := failingService(t, &flaky{err: cause, fails: 100})
			if e := svc.ProcessNext(context.Background(), time.Now()); !errors.Is(e, cause) {
				t.Fatalf("the worker must see the cause: %v", e)
			}
			if got, want := captureEntryText(t, db), domain.ReasonOf(cause).Sentence(); got != want {
				t.Fatalf("entry text %q, want %q", got, want)
			}
		})
	}
}

func TestRetryRunsTheExtractionOnceMore(t *testing.T) {
	x := &flaky{err: &domain.ModelError{Message: "x", Reason: domain.ReasonCutOff}, fails: 1}
	svc, _, id := failingService(t, x)
	if e := svc.ProcessNext(context.Background(), time.Now()); !errors.Is(e, x.err) {
		t.Fatalf("the worker must see the cause: %v", e)
	}
	if _, e := svc.Retry(id); e != nil {
		t.Fatal(e)
	}
	if e := svc.ProcessNext(context.Background(), time.Now()); e != nil {
		t.Fatal(e)
	}
	if c, _ := svc.Get(id); x.calls != 2 || c.Status == "failed" || len(c.Candidates) != 1 {
		t.Fatalf("calls %d status %s candidates %d", x.calls, c.Status, len(c.Candidates))
	}
}
