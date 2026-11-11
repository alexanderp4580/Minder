package reminder

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
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
	w := Worker{Store: s, Sender: f}
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
	s.ReminderAction(h[0].Reminder.ID, "ack", now)
	w.Tick(context.Background(), now.Add(time.Hour))
	if f.calls != 2 {
		t.Fatal("acknowledged reminder sent")
	}
}
func TestFCMPayloadAndInvalidTokenClassification(t *testing.T) {
	status := 200
	detail := "UNREGISTERED"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-access" {
			t.Error("missing authorization")
		}
		var data struct {
			Message struct {
				Token   string            `json:"token"`
				Data    map[string]string `json:"data"`
				Android struct {
					Priority string `json:"priority"`
				} `json:"android"`
			} `json:"message"`
		}
		if e := json.NewDecoder(r.Body).Decode(&data); e != nil {
			t.Error(e)
		}
		if data.Message.Token != "device" || data.Message.Data["reminder_id"] != "r" || data.Message.Android.Priority != "HIGH" {
			t.Error("invalid payload")
		}
		w.WriteHeader(status)
		if status != 200 {
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"details": []any{map[string]string{"@type": "type.googleapis.com/google.firebase.fcm.v1.FcmError", "errorCode": detail}}}})
		} else {
			w.Write([]byte(`{"name":"projects/test/messages/1"}`))
		}
	}))
	defer server.Close()
	f := FCM{ProjectID: "test", Client: server.Client(), Endpoint: server.URL, AccessToken: func(context.Context) (string, error) { return "test-access", nil }}
	invalid, e := f.Send(context.Background(), "device", map[string]string{"reminder_id": "r"})
	if e != nil || invalid {
		t.Fatal(e)
	}
	status = 404
	invalid, e = f.Send(context.Background(), "device", map[string]string{"reminder_id": "r"})
	if e == nil || !invalid {
		t.Fatal("invalid token not classified")
	}
	status = 503
	detail = "UNAVAILABLE"
	invalid, e = f.Send(context.Background(), "device", map[string]string{"reminder_id": "r"})
	if e == nil || invalid {
		t.Fatal("transient failure deleted token")
	}
}
