package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/config"
)

func TestLoggerReadsLevelAndFormatFromTheEnvironment(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	var buf bytes.Buffer
	logger, err := newLogger(env(map[string]string{"LOG_LEVEL": "warn", "LOG_FORMAT": "json"}), &buf)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hidden")
	logger.Warn("shown", "k", "v")
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil || record["msg"] != "shown" || record["k"] != "v" {
		t.Fatalf("json output wrong: %q %v", buf.String(), err)
	}
	buf.Reset()
	logger, _ = newLogger(env(nil), &buf)
	logger.Debug("hidden")
	logger.Info("shown")
	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "msg=shown") {
		t.Fatalf("default is info level, text format: %q", buf.String())
	}
	if _, err = newLogger(env(map[string]string{"LOG_LEVEL": "loud"}), &buf); err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("bad level accepted: %v", err)
	}
	if _, err = newLogger(env(map[string]string{"LOG_FORMAT": "xml"}), &buf); err == nil || !strings.Contains(err.Error(), "LOG_FORMAT") {
		t.Fatalf("bad format accepted: %v", err)
	}
}

func TestStartupSummaryShowsSettingsButNeverSecrets(t *testing.T) {
	settings := config.Settings{
		DatabasePath: "/data/minder.db", WebDir: "/app/web", ListenAddr: "0.0.0.0:8080",
		ModelKey: "sk-SECRETMODELKEY", Model: "anthropic/x", MonthlyLimitEUR: 5, USDToEUR: 0.92,
		SerperKey: "SECRETSERPERKEY", FirebaseServiceAccount: `{"private_key":"SECRETFIREBASE"}`, FirebaseAndroidConfig: `{"api_key":"PUBLICKEY"}`,
	}
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logStartup(logger, settings, "cloudflare")
	var r map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &r); err != nil {
		t.Fatal(err)
	}
	if r["msg"] != "server configuration" || r["database_path"] != "/data/minder.db" || r["model_key_set"] != true || r["serper_key_set"] != true || r["firebase_service_account_set"] != true || r["auth_mode"] != "cloudflare" || r["monthly_limit_eur"] != float64(5) || r["model"] != "anthropic/x" || r["pid"] == nil {
		t.Fatalf("summary incomplete: %v", r)
	}
	if strings.Contains(buf.String(), "SECRET") || strings.Contains(buf.String(), "PUBLICKEY") {
		t.Fatalf("secret reached the summary: %s", buf.String())
	}
}

func TestShutdownIsLogged(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(previous)
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, &http.Server{}, listener, func(ctx context.Context) { <-ctx.Done() }) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not stop")
	}
	out := buf.String()
	for _, want := range []string{"shutdown requested", "http server stopped", "background worker stopped"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}
