package config

import (
	"strings"
	"testing"
	"time"
)

func TestAgentTraceSettings(t *testing.T) {
	for _, tc := range []struct {
		name, dir, until string
		invalid          bool
	}{
		{"disabled", "", "", false},
		{"enabled", "/data/agent-traces", "2026-11-16T12:00:00Z", false},
		{"expired", "/data/agent-traces", "2020-01-01T00:00:00Z", false},
		{"missing expiry", "/data/agent-traces", "", true},
		{"missing directory", "", "2026-11-16T12:00:00Z", true},
		{"bad expiry", "/data/agent-traces", "tomorrow", true},
		{"relative directory", "agent-traces", "2026-11-16T12:00:00Z", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Load(func(key string) string {
				if key == "AGENT_TRACE_DIR" {
					return tc.dir
				}
				if key == "AGENT_TRACE_UNTIL" {
					return tc.until
				}
				return ""
			})
			if (err != nil) != tc.invalid {
				t.Fatalf("validation: %v", err)
			}
			if err != nil {
				if !strings.Contains(err.Error(), "AGENT_TRACE") || strings.Contains(err.Error(), tc.dir) && tc.dir != "" {
					t.Fatal("unsafe or unclear setting error")
				}
				return
			}
			if cfg.AgentTraceDir != tc.dir {
				t.Fatal("directory ignored")
			}
			if tc.until != "" && cfg.AgentTraceUntil.Format(time.RFC3339) != tc.until {
				t.Fatal("expiry ignored")
			}
		})
	}
}

func TestLoadRejectsInvalidBudget(t *testing.T) {
	for _, key := range []string{"MONTHLY_LIMIT_EUR", "USD_TO_EUR"} {
		for _, value := range []string{"NaN", "+Inf", "-Inf", "0", "-1", "bad"} {
			t.Run(key+"/"+value, func(t *testing.T) {
				_, err := Load(func(k string) string {
					if k == key {
						return value
					}
					return ""
				})
				if err == nil || !strings.Contains(err.Error(), key) {
					t.Fatalf("want %s validation, got %v", key, err)
				}
			})
		}
	}
}

func TestLoadBudget(t *testing.T) {
	cfg, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MonthlyLimitEUR != 5 || cfg.USDToEUR != 1 {
		t.Fatalf("unexpected defaults: %v, %v", cfg.MonthlyLimitEUR, cfg.USDToEUR)
	}
	cfg, err = Load(func(key string) string {
		switch key {
		case "MONTHLY_LIMIT_EUR":
			return "2.5"
		case "USD_TO_EUR":
			return "0.9"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MonthlyLimitEUR != 2.5 || cfg.USDToEUR != 0.9 {
		t.Fatalf("settings ignored: %v, %v", cfg.MonthlyLimitEUR, cfg.USDToEUR)
	}
}

func TestLoadModelEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, environment, url string
		wantError              bool
	}{
		{name: "production default"},
		{name: "production explicit", url: "https://openrouter.ai/api/v1/chat/completions"},
		{name: "production custom", url: "http://localhost:8181", wantError: true},
		{name: "development fixture", environment: "development", url: "http://localhost:8181"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(func(key string) string {
				switch key {
				case "ENV":
					return tc.environment
				case "OPENROUTER_URL":
					return tc.url
				}
				return ""
			})
			if (err != nil) != tc.wantError {
				t.Fatalf("want error %v, got %v", tc.wantError, err)
			}
		})
	}
}

func TestLoadRejectsUnknownEnvironment(t *testing.T) {
	_, err := Load(func(key string) string {
		if key == "ENV" {
			return "prodution"
		}
		return ""
	})
	if err == nil || !strings.Contains(err.Error(), "ENV") {
		t.Fatalf("want ENV error, got %v", err)
	}
}

func TestSerperSettings(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}
	off, err := Load(env(nil))
	if err != nil || off.SerperKey != "" || off.SerperEndpoint != "" {
		t.Fatalf("default %+v %v", off, err)
	}
	on, err := Load(env(map[string]string{"SERPER_API_KEY": "k", "SERPER_URL": "https://google.serper.dev"}))
	if err != nil || on.SerperKey != "k" || on.SerperEndpoint != "https://google.serper.dev" {
		t.Fatalf("set %+v %v", on, err)
	}
	_, err = Load(env(map[string]string{"SERPER_API_KEY": "k", "SERPER_URL": "https://google.serper.dev/maps"}))
	if err == nil || !strings.Contains(err.Error(), "SERPER_URL") {
		t.Fatalf("a path on the base URL %v", err)
	}
	_, err = Load(env(map[string]string{"SERPER_API_KEY": "secret-value", "SERPER_URL": "http://example.test"}))
	if err == nil || !strings.Contains(err.Error(), "SERPER_URL") || strings.Contains(err.Error(), "secret-value") || strings.Contains(err.Error(), "example.test") {
		t.Fatalf("production override %v", err)
	}
	if _, err = Load(env(map[string]string{"ENV": "test", "SERPER_URL": "http://127.0.0.1:1"})); err != nil {
		t.Fatalf("test override %v", err)
	}
}
