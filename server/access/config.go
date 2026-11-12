package access

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Authenticator checks the identity attached to a request.
type Authenticator interface {
	Authenticate(*http.Request) error
}

// FromEnvironment loads the authentication adapter. Development and tests can omit authentication.
func FromEnvironment(getenv func(string) string) (Authenticator, error) {
	mode := getenv("AUTH_MODE")
	if mode == "" {
		mode = "cloudflare"
	}
	if mode == "dev" {
		if environment := getenv("ENV"); environment != "development" && environment != "test" {
			return nil, fmt.Errorf("dev authentication is forbidden in production or an unknown environment")
		}
		return nil, nil
	}
	if mode != "cloudflare" {
		return nil, fmt.Errorf("AUTH_MODE must be dev or cloudflare")
	}
	values := make(map[string]string, 3)
	for _, key := range []string{"CF_ISSUER", "CF_AUDIENCE", "OWNER_EMAIL"} {
		value := strings.TrimSpace(getenv(key))
		if value == "" {
			return nil, fmt.Errorf("%s is required", key)
		}
		values[key] = value
	}
	issuer := strings.TrimSuffix(values["CF_ISSUER"], "/")
	u, err := url.Parse(issuer)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("CF_ISSUER must be an HTTPS origin")
	}
	return &cloudflareVerifier{Issuer: issuer, Audience: values["CF_AUDIENCE"], Email: values["OWNER_EMAIL"]}, nil
}
