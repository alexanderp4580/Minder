package api

import (
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/domain"
	"io"
	"net/http"
	"strings"
	"time"
)

// PushConfig contains Android public identifiers only, never the service account.
type PushConfig struct {
	AppID     string `json:"app_id"`
	APIKey    string `json:"api_key"`
	ProjectID string `json:"project_id"`
	SenderID  string `json:"sender_id"`
}

func ParsePushConfig(raw string) (*PushConfig, error) {
	if raw == "" {
		return nil, nil
	}
	var config PushConfig
	if e := json.Unmarshal([]byte(raw), &config); e != nil {
		return nil, e
	}
	if config.AppID == "" || config.APIKey == "" || config.ProjectID == "" || config.SenderID == "" {
		return nil, &domain.Invalid{Field: "push", Message: "Firebase Android configuration is incomplete."}
	}
	return &config, nil
}
func decodeBody(w http.ResponseWriter, r *http.Request, v any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return &domain.Invalid{Field: "request", Message: "Send a valid request."}
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return &domain.Invalid{Field: "request", Message: "Send one request."}
	}
	return nil
}
func (s *Server) reminderRoutes(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case r.Method == "GET" && r.URL.Path == "/v1/reminders":
		entries, e := s.store.ActiveReminders()
		if e != nil {
			fail(w, e)
		} else {
			write(w, 200, entries)
		}
		return true
	case r.Method == "GET" && r.URL.Path == "/v1/push/config":
		if s.PushConfig == nil {
			write(w, 503, map[string]string{"code": "push_not_configured", "message": "Push setup is not ready on the server."})
		} else {
			write(w, 200, s.PushConfig)
		}
		return true
	case r.Method == "POST" && r.URL.Path == "/v1/push/devices":
		var input struct {
			DeviceID string `json:"device_id"`
			Token    string `json:"token"`
		}
		e := decodeBody(w, r, &input)
		if e == nil {
			e = s.store.RegisterDevice(input.DeviceID, input.Token)
		}
		if e != nil {
			fail(w, e)
		} else {
			write(w, 200, map[string]bool{"registered": true})
		}
		return true
	case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/v1/reminders/"):
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/reminders/"), "/")
		if len(parts) != 2 || parts[0] == "" {
			http.NotFound(w, r)
			return true
		}
		entry, e := s.store.ReminderAction(parts[0], parts[1], time.Now().UTC())
		if e != nil {
			fail(w, e)
		} else {
			write(w, 200, entry)
		}
		return true
	}
	return false
}
