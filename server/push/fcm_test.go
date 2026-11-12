package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
