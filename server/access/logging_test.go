package access

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// syncBuffer lets the test read log output while background goroutines still write it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func capture(t *testing.T) func() []map[string]any {
	t.Helper()
	var buf syncBuffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return func() []map[string]any {
		var out []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			var r map[string]any
			if json.Unmarshal([]byte(line), &r) == nil {
				out = append(out, r)
			}
		}
		return out
	}
}
func first(records []map[string]any, msg string) map[string]any {
	for _, r := range records {
		if r["msg"] == msg {
			return r
		}
	}
	return nil
}

func TestRejectedAuthenticationLogsReasonButNeverTheToken(t *testing.T) {
	logs := capture(t)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := &cloudflareVerifier{Issuer: "https://team.cloudflareaccess.com", Audience: "aud", Email: "owner@example.com", Keys: map[string]*rsa.PublicKey{"key": &key.PublicKey}}
	sign := func(claims jwt.MapClaims) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = "key"
		s, _ := tok.SignedString(key)
		return s
	}
	cases := []struct{ name, token, reason, detail string }{
		{"missing", "", "missing_token", ""},
		{"expired", sign(jwt.MapClaims{"iss": v.Issuer, "aud": []string{"aud"}, "email": v.Email, "exp": time.Now().Add(-time.Hour).Unix()}), "invalid_token", "expired"},
		{"owner", sign(jwt.MapClaims{"iss": v.Issuer, "aud": []string{"aud"}, "email": "intruder@evil.example", "exp": time.Now().Add(time.Hour).Unix()}), "wrong_owner", "evil.example"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/v1/chat", nil)
		r.Header.Set("Cf-Access-Jwt-Assertion", c.token)
		if v.Authenticate(r) == nil {
			t.Fatalf("%s accepted", c.name)
		}
		rec := first(logs(), "authentication rejected")
		var matched map[string]any
		for _, x := range logs() {
			if x["msg"] == "authentication rejected" && x["reason"] == c.reason {
				matched = x
			}
		}
		if matched == nil || matched["level"] != "WARN" {
			t.Fatalf("%s: reason %s not logged (%v)", c.name, c.reason, rec)
		}
		if c.detail != "" && !strings.Contains(fmt.Sprint(matched["detail"]), c.detail) {
			t.Fatalf("%s: detail %v lacks %q", c.name, matched["detail"], c.detail)
		}
		if c.token != "" {
			for _, line := range logs() {
				b, _ := json.Marshal(line)
				if strings.Contains(string(b), c.token) || strings.Contains(string(b), "intruder@") {
					t.Fatalf("token or full email reached the log: %s", b)
				}
			}
		}
	}
}

func TestAcceptedAuthenticationIsLoggedAtDebug(t *testing.T) {
	logs := capture(t)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := &cloudflareVerifier{Issuer: "https://team.cloudflareaccess.com", Audience: "aud", Email: "owner@example.com", Keys: map[string]*rsa.PublicKey{"key": &key.PublicKey}}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": v.Issuer, "aud": []string{"aud"}, "email": v.Email, "exp": time.Now().Add(time.Hour).Unix()})
	tok.Header["kid"] = "key"
	signed, _ := tok.SignedString(key)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Cf-Access-Jwt-Assertion", signed)
	if err := v.Authenticate(r); err != nil {
		t.Fatal(err)
	}
	if rec := first(logs(), "authentication accepted"); rec == nil || rec["level"] != "DEBUG" {
		t.Fatalf("acceptance not logged at debug: %v", rec)
	}
}

func TestSigningKeyFetchIsLogged(t *testing.T) {
	logs := capture(t)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"Kid": "k1", "N": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "E": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	defer srv.Close()
	v := &cloudflareVerifier{Issuer: srv.URL, Audience: "aud", Email: "o@example.com", Client: srv.Client()}
	if _, err := v.key("k1"); err != nil {
		t.Fatal(err)
	}
	rec := first(logs(), "signing keys fetched")
	if rec == nil || rec["keys"] != float64(1) || rec["status"] != float64(200) || rec["kid_found"] != true {
		t.Fatalf("key fetch not logged: %v", rec)
	}
	if _, err := v.key("missing"); err == nil {
		t.Fatal("unknown key accepted")
	}
	if first(logs(), "signing key lookup failed") == nil {
		t.Fatal("unknown key not logged")
	}
}
