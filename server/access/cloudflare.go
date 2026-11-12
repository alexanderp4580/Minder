package access

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type cloudflareVerifier struct {
	Issuer, Audience, Email string
	Keys                    map[string]*rsa.PublicKey
	mu                      sync.Mutex
	fetched                 time.Time
	Client                  *http.Client
}

func (v *cloudflareVerifier) Authenticate(r *http.Request) error {
	return v.Verify(r.Header.Get("Cf-Access-Jwt-Assertion"))
}

func (v *cloudflareVerifier) Verify(raw string) error {
	if raw == "" {
		slog.Warn("authentication rejected", "reason", "missing_token")
		return errors.New("missing token")
	}
	claims := jwt.MapClaims{}
	token, e := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return v.key(kid)
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(v.Issuer),
		jwt.WithAudience(v.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if e != nil || !token.Valid {
		detail := "token not valid"
		if e != nil {
			detail = e.Error()
		}
		slog.Warn("authentication rejected", "reason", "invalid_token", "detail", detail)
		return errors.New("invalid token")
	}
	email, _ := claims["email"].(string)
	if email == "" || !strings.EqualFold(email, v.Email) {
		slog.Warn("authentication rejected", "reason", "wrong_owner", "detail", "email domain "+emailDomain(email))
		return errors.New("wrong owner")
	}
	slog.Debug("authentication accepted", "key_id", token.Header["kid"], "expires", claims["exp"])
	return nil
}

// emailDomain keeps logs free of mailbox names.
func emailDomain(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		return email[i+1:]
	}
	return "(none)"
}
func (v *cloudflareVerifier) key(kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if key := v.Keys[kid]; key != nil && (v.fetched.IsZero() || time.Since(v.fetched) < time.Hour) {
		return key, nil
	}
	if time.Since(v.fetched) < time.Minute {
		slog.Warn("signing key lookup failed", "reason", "key unknown and keys were fetched less than a minute ago", "key_id", kid)
		return nil, errors.New("unknown key")
	}
	started := time.Now()
	v.fetched = started
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	res, e := client.Get(v.Issuer + "/cdn-cgi/access/certs")
	if e != nil {
		slog.Error("signing key fetch failed", "error", e, "duration_ms", time.Since(started).Milliseconds())
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		slog.Error("signing key fetch failed", "status", res.StatusCode, "duration_ms", time.Since(started).Milliseconds())
		return nil, errors.New("keys unavailable")
	}
	var jwks struct{ Keys []struct{ Kid, N, E string } }
	if e = json.NewDecoder(http.MaxBytesReader(nil, res.Body, 1<<20)).Decode(&jwks); e != nil {
		slog.Error("signing key fetch failed", "reason", "unreadable key set", "error", e)
		return nil, e
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range jwks.Keys {
		n, en := base64.RawURLEncoding.DecodeString(k.N)
		ex, ee := base64.RawURLEncoding.DecodeString(k.E)
		if en != nil || ee != nil || len(ex) > 4 {
			continue
		}
		value := 0
		for _, b := range ex {
			value = value*256 + int(b)
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: value}
	}
	v.Keys = keys
	_, found := keys[kid]
	slog.Info("signing keys fetched", "status", res.StatusCode, "keys", len(keys), "listed", len(jwks.Keys), "kid_found", found, "duration_ms", time.Since(started).Milliseconds())
	if k := keys[kid]; k != nil {
		return k, nil
	}
	slog.Warn("signing key lookup failed", "reason", "key not in fetched set", "key_id", kid)
	return nil, errors.New("unknown signing key")
}
