package config

import (
	"strings"
	"testing"
)

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
	on, err := Load(env(map[string]string{"SERPER_API_KEY": "k", "SERPER_URL": "https://google.serper.dev/places"}))
	if err != nil || on.SerperKey != "k" || on.SerperEndpoint != "https://google.serper.dev/places" {
		t.Fatalf("set %+v %v", on, err)
	}
	_, err = Load(env(map[string]string{"SERPER_API_KEY": "secret-value", "SERPER_URL": "http://example.test"}))
	if err == nil || !strings.Contains(err.Error(), "SERPER_URL") || strings.Contains(err.Error(), "secret-value") || strings.Contains(err.Error(), "example.test") {
		t.Fatalf("production override %v", err)
	}
	if _, err = Load(env(map[string]string{"ENV": "test", "SERPER_URL": "http://127.0.0.1:1"})); err != nil {
		t.Fatalf("test override %v", err)
	}
}
