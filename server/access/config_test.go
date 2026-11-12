package access

import (
	"strings"
	"testing"
)

func TestFromEnvironment(t *testing.T) {
	valid := map[string]string{"CF_ISSUER": "https://team.cloudflareaccess.com", "CF_AUDIENCE": "audience", "OWNER_EMAIL": "owner@example.com"}
	for _, tc := range []struct {
		name, key, value, wantError, environment string
		dev                                      bool
	}{
		{name: "configured"},
		{name: "missing issuer", key: "CF_ISSUER", wantError: "CF_ISSUER"},
		{name: "missing audience", key: "CF_AUDIENCE", wantError: "CF_AUDIENCE"},
		{name: "blank owner", key: "OWNER_EMAIL", value: "  ", wantError: "OWNER_EMAIL"},
		{name: "insecure issuer", key: "CF_ISSUER", value: "http://team.cloudflareaccess.com", wantError: "CF_ISSUER"},
		{name: "issuer credentials", key: "CF_ISSUER", value: "https://user:password@team.cloudflareaccess.com", wantError: "CF_ISSUER"},
		{name: "issuer path", key: "CF_ISSUER", value: "https://team.cloudflareaccess.com/path", wantError: "CF_ISSUER"},
		{name: "unknown mode", key: "AUTH_MODE", value: "other", wantError: "AUTH_MODE"},
		{name: "production dev", key: "AUTH_MODE", value: "dev", wantError: "production"},
		{name: "unknown environment dev", key: "AUTH_MODE", value: "dev", wantError: "unknown environment", environment: "prodution"},
		{name: "local dev", key: "AUTH_MODE", value: "dev", dev: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]string{}
			for k, v := range valid {
				values[k] = v
			}
			if tc.dev {
				values["ENV"] = "development"
				delete(values, "CF_ISSUER")
				delete(values, "CF_AUDIENCE")
				delete(values, "OWNER_EMAIL")
			}
			if tc.environment != "" {
				values["ENV"] = tc.environment
			}
			if tc.key != "" {
				values[tc.key] = tc.value
			}
			verifier, err := FromEnvironment(func(key string) string { return values[key] })
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("want error containing %q, got %v", tc.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.dev {
				if verifier != nil {
					t.Fatal("dev requires no verifier")
				}
				return
			}
			if verifier == nil {
				t.Fatal("authentication is disabled")
			}
			identity := verifier.(*cloudflareVerifier)
			if identity.Issuer != valid["CF_ISSUER"] || identity.Audience != valid["CF_AUDIENCE"] || identity.Email != valid["OWNER_EMAIL"] {
				t.Fatal("configured identity was not preserved")
			}
		})
	}
}
