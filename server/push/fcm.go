package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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
	access, e := f.AccessToken(ctx)
	if e != nil {
		return false, errors.New("Firebase authorization failed")
	}
	endpoint := f.Endpoint
	if endpoint == "" {
		endpoint = "https://fcm.googleapis.com/v1/projects/" + f.ProjectID + "/messages:send"
	}
	body, _ := json.Marshal(map[string]any{"message": map[string]any{"token": token, "data": data, "android": map[string]string{"priority": "HIGH", "ttl": "86400s"}}})
	req, e := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if e != nil {
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
		return false, errors.New("Firebase delivery failed")
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 65536))
	if e != nil {
		return false, errors.New("Firebase response failed")
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return false, nil
	}
	var failure struct {
		Error struct {
			Details []struct {
				Type string `json:"@type"`
				Code string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	json.Unmarshal(raw, &failure)
	invalid := false
	for _, d := range failure.Error.Details {
		if strings.HasSuffix(d.Type, "google.firebase.fcm.v1.FcmError") && d.Code == "UNREGISTERED" {
			invalid = true
		}
	}
	return invalid, errors.New("Firebase did not accept delivery")
}
