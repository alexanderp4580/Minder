// Package config loads process settings. Authentication settings belong to access.
package config

import (
	"fmt"
	"math"
	"strconv"
)

type Settings struct {
	Email                                         EmailSettings
	DatabasePath, WebDir, BrandDir, ListenAddr    string
	ModelKey, Model, ModelEndpoint                string
	MonthlyLimitEUR, USDToEUR                     float64
	FirebaseServiceAccount, FirebaseAndroidConfig string
}

func Load(getenv func(string) string) (Settings, error) {
	value := func(key, fallback string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return fallback
	}
	environment := value("ENV", "production")
	switch environment {
	case "production", "development", "test":
	default:
		return Settings{}, fmt.Errorf("ENV must be production, development or test")
	}
	cfg := Settings{
		DatabasePath:           value("DATABASE_PATH", "minder.db"),
		WebDir:                 value("WEB_DIR", "../app/build/web"),
		BrandDir:               getenv("BRAND_DIR"),
		ListenAddr:             value("LISTEN_ADDR", "127.0.0.1:8080"),
		ModelKey:               getenv("OPENROUTER_API_KEY"),
		Model:                  getenv("MODEL"),
		ModelEndpoint:          getenv("OPENROUTER_URL"),
		FirebaseServiceAccount: getenv("FIREBASE_SERVICE_ACCOUNT_JSON"),
		FirebaseAndroidConfig:  getenv("FIREBASE_ANDROID_CONFIG"),
	}
	if environment == "production" && cfg.ModelEndpoint != "" && cfg.ModelEndpoint != "https://openrouter.ai/api/v1/chat/completions" {
		return Settings{}, fmt.Errorf("OPENROUTER_URL cannot override the production endpoint")
	}
	var err error
	cfg.MonthlyLimitEUR, err = positiveNumber(value("MONTHLY_LIMIT_EUR", "5"), "MONTHLY_LIMIT_EUR")
	if err != nil {
		return Settings{}, err
	}
	cfg.USDToEUR, err = positiveNumber(value("USD_TO_EUR", "1"), "USD_TO_EUR")
	if err != nil {
		return Settings{}, err
	}
	cfg.Email, err = LoadEmail(getenv)
	if err != nil {
		return Settings{}, err
	}
	return cfg, nil
}

func positiveNumber(raw, key string) (float64, error) {
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || n <= 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("%s must be a finite positive number", key)
	}
	return n, nil
}
