package push

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type FCM struct {
	ProjectID   string
	Client      *http.Client
	Endpoint    string
	AccessToken func(context.Context) (string, error)
}

func NewFCM(raw string) (*FCM, error) {
	config, e := google.JWTConfigFromJSON([]byte(raw), "https://www.googleapis.com/auth/firebase.messaging")
	if e != nil {
		return nil, errors.New("Firebase service account is invalid")
	}
	var meta struct {
		ProjectID string `json:"project_id"`
	}
	if e = json.Unmarshal([]byte(raw), &meta); e != nil || !regexp.MustCompile(`^[a-z][a-z0-9-]{4,61}[a-z0-9]$`).MatchString(meta.ProjectID) {
		return nil, errors.New("Firebase project ID is invalid")
	}
	// Never let a supplied key redirect signed credentials to another endpoint.
	if config.TokenURL != "https://oauth2.googleapis.com/token" {
		return nil, errors.New("Firebase token endpoint is invalid")
	}
	tokenContext := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: 10 * time.Second})
	source := config.TokenSource(tokenContext)
	return &FCM{ProjectID: meta.ProjectID, Client: &http.Client{}, AccessToken: func(ctx context.Context) (string, error) {
		t, e := source.Token()
		if e != nil {
			return "", errors.New("Firebase authorization failed")
		}
		return t.AccessToken, nil
	}}, nil
}
func (f *FCM) Send(ctx context.Context, token string, data map[string]string) (bool, error) {
	started := time.Now()
	fingerprint := fingerprint(token)
	access, e := f.AccessToken(ctx)
	if e != nil {
		slog.Error("push failed", "stage", "authorize", "error", e, "token_fp", fingerprint, "reminder_id", data["reminder_id"], "project_id", f.ProjectID)
		return false, errors.New("Firebase authorization failed")
	}
	endpoint := f.Endpoint
	if endpoint == "" {
		endpoint = "https://fcm.googleapis.com/v1/projects/" + f.ProjectID + "/messages:send"
	}
	body, _ := json.Marshal(map[string]any{"message": map[string]any{"token": token, "data": data, "android": map[string]string{"priority": "HIGH", "ttl": "86400s"}}})
	req, e := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if e != nil {
		slog.Error("push failed", "stage", "build", "error", e, "token_fp", fingerprint)
		return false, errors.New("Firebase request failed")
	}
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	client := f.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, e := client.Do(req)
	if e != nil {
		slog.Error("push failed", "stage", "connect", "error", e, "token_fp", fingerprint, "reminder_id", data["reminder_id"], "duration_ms", time.Since(started).Milliseconds())
		return false, errors.New("Firebase delivery failed")
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 65536))
	if e != nil {
		slog.Error("push failed", "stage", "read response", "error", e, "status", res.StatusCode, "token_fp", fingerprint)
		return false, errors.New("Firebase response failed")
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		var accepted struct {
			Name string `json:"name"`
		}
		json.Unmarshal(raw, &accepted)
		slog.Info("push sent", "status", res.StatusCode, "message_name", accepted.Name, "token_fp", fingerprint, "reminder_id", data["reminder_id"], "late", data["late"], "project_id", f.ProjectID, "duration_ms", time.Since(started).Milliseconds())
		return false, nil
	}
	var failure struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Details []struct {
				Type string `json:"@type"`
				Code string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	json.Unmarshal(raw, &failure)
	invalid := false
	fcmCode := ""
	for _, d := range failure.Error.Details {
		if strings.HasSuffix(d.Type, "google.firebase.fcm.v1.FcmError") {
			fcmCode = d.Code
			if d.Code == "UNREGISTERED" {
				invalid = true
			}
		}
	}
	slog.Warn("push rejected", "status", res.StatusCode, "fcm_status", failure.Error.Status, "fcm_message", failure.Error.Message, "fcm_error_code", fcmCode, "token_invalid", invalid, "token_fp", fingerprint, "reminder_id", data["reminder_id"], "duration_ms", time.Since(started).Milliseconds())
	return invalid, errors.New("Firebase did not accept delivery")
}

// fingerprint names a push token in logs without revealing it. It matches domain.Fingerprint, which this adapter may not import.
func fingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:4])
}
