package reminder

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

type fakePush struct {
	calls         int
	fail, invalid bool
}

func (f *fakePush) Send(ctx context.Context, token string, data map[string]string) (bool, error) {
	f.calls++
	if f.invalid {
		return true, errors.New("invalid")
	}
	if f.fail {
		return false, errors.New("retry")
	}
	return false, nil
}
func TestWorkerRetriesAndStopsAfterAcknowledgement(t *testing.T) {
	s, e := store.Open(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	now := time.Now().UTC().Truncate(time.Second)
	s.Write(func(tx *store.Tx) error {
		return tx.PutItem(domain.Item{ID: "x", Title: "Bins", Status: "open", Version: 1, RemindAt: now.Format(time.RFC3339), NagMinutes: 15})
	})
	s.RegisterDevice("p", "token")
	f := &fakePush{fail: true}
	w := Worker{Repository: s, Scheduler: New(s), Sender: f}
	if e = w.Tick(context.Background(), now); e != nil {
		t.Fatal(e)
	}
	if f.calls != 1 {
		t.Fatal("not sent")
	}
	w.Tick(context.Background(), now.Add(30*time.Second))
	if f.calls != 1 {
		t.Fatal("early retry")
	}
	f.fail = false
	w.Tick(context.Background(), now.Add(time.Minute))
	if f.calls != 2 {
		t.Fatal("retry lost")
	}
	h, _ := s.ActiveReminders()
	New(s).Act(h[0].Reminder.ID, "ack", now)
	w.Tick(context.Background(), now.Add(time.Hour))
	if f.calls != 2 {
		t.Fatal("acknowledged reminder sent")
	}
}
