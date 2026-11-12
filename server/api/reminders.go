package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/alexanderp4580/Minder/server/domain"
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
func (s *Server) reminders(w http.ResponseWriter, r *http.Request) {
	entries, e := s.Reminders.Active()
	if e != nil {
		fail(w, e)
		return
	}
	note(w, "reminders", len(entries))
	write(w, http.StatusOK, entries)
}
func (s *Server) pushConfig(w http.ResponseWriter, r *http.Request) {
	note(w, "push_configured", s.PushConfig != nil)
	if s.PushConfig == nil {
		write(w, http.StatusServiceUnavailable, map[string]string{"code": "push_not_configured", "message": "Push setup is not ready on the server."})
		return
	}
	write(w, http.StatusOK, s.PushConfig)
}
func (s *Server) registerDevice(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DeviceID string `json:"device_id"`
		Token    string `json:"token"`
	}
	e := decodeBody(w, r, &input)
	note(w, "device_id", input.DeviceID, "token_len", len(input.Token), "token_fp", domain.Fingerprint(input.Token))
	if e == nil {
		e = s.Reminders.RegisterDevice(input.DeviceID, input.Token)
	}
	if e != nil {
		fail(w, e)
		return
	}
	write(w, http.StatusOK, map[string]bool{"registered": true})
}
func (s *Server) reminderAction(w http.ResponseWriter, r *http.Request) {
	note(w, "reminder_id", r.PathValue("id"), "action", r.PathValue("action"))
	entry, e := s.Reminders.Act(r.PathValue("id"), r.PathValue("action"), time.Now().UTC())
	if e != nil {
		fail(w, e)
		return
	}
	note(w, "entry_id", entry.ID)
	write(w, http.StatusOK, entry)
}
