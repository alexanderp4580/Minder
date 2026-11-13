package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/alexanderp4580/Minder/server/config"
)

// newLogger builds the process logger. LOG_LEVEL is debug, info, warn or error. LOG_FORMAT is text or json.
func newLogger(getenv func(string) string, w io.Writer) (*slog.Logger, error) {
	var level slog.Level
	switch name := getenv("LOG_LEVEL"); name {
	case "", "info":
		level = slog.LevelInfo
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error")
	}
	options := &slog.HandlerOptions{Level: level}
	switch format := getenv("LOG_FORMAT"); format {
	case "", "text":
		return slog.New(slog.NewTextHandler(w, options)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(w, options)), nil
	default:
		return nil, fmt.Errorf("LOG_FORMAT must be text or json")
	}
}

// logStartup records the effective settings. It reports whether each secret is set, never its value.
func logStartup(logger *slog.Logger, s config.Settings, authMode string) {
	revision, modified := "unknown", false
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				revision = kv.Value
			case "vcs.modified":
				modified = kv.Value == "true"
			}
		}
	}
	host, _ := os.Hostname()
	logger.Info("server configuration",
		"pid", os.Getpid(), "host", host, "go", runtime.Version(), "revision", revision, "dirty", modified,
		"listen_addr", s.ListenAddr, "database_path", s.DatabasePath, "web_dir", s.WebDir, "brand_dir", s.BrandDir,
		"auth_mode", authMode, "model", s.Model, "model_endpoint", s.ModelEndpoint, "model_key_set", s.ModelKey != "", "serper_key_set", s.SerperKey != "",
		"monthly_limit_eur", s.MonthlyLimitEUR, "usd_to_eur", s.USDToEUR,
		"firebase_service_account_set", s.FirebaseServiceAccount != "", "firebase_android_config_set", s.FirebaseAndroidConfig != "",
	)
}
