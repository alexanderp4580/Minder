package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEmailConfigurationFailsClosed(t *testing.T) {
	valid := map[string]string{"EMAIL_CAPTURE_ENABLED": "true", "EMAIL_ALLOWED_SENDERS": "one@example.com, Two@example.net", "EMAIL_RECIPIENT": "assistant@example.org", "EMAIL_INGEST_HMAC_KEY": base64.StdEncoding.EncodeToString(make([]byte, 32)), "EMAIL_SOURCE_DIR": "/private/sources"}
	for _, key := range []string{"EMAIL_ALLOWED_SENDERS", "EMAIL_RECIPIENT", "EMAIL_INGEST_HMAC_KEY", "EMAIL_SOURCE_DIR"} {
		t.Run(key, func(t *testing.T) {
			_, e := LoadEmail(func(k string) string {
				if k == key {
					return ""
				}
				return valid[k]
			})
			if e == nil || strings.Contains(e.Error(), "one@example.com") {
				t.Fatalf("unsafe validation: %v", e)
			}
		})
	}
	for _, raw := range []string{"", "one@example.com,", "Person <one@example.com>", "one@example.com;two@example.net"} {
		if _, e := LoadEmail(func(k string) string {
			if k == "EMAIL_ALLOWED_SENDERS" {
				return raw
			}
			return valid[k]
		}); e == nil {
			t.Fatal("invalid allowlist accepted")
		}
	}
	cfg, e := LoadEmail(func(k string) string { return valid[k] })
	if e != nil || len(cfg.AllowedSenders) != 2 || cfg.AllowedSenders[1] != "two@example.net" {
		t.Fatal(cfg, e)
	}
}
