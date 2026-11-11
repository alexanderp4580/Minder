package api

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

func ValidateMode(mode, env string) error {
	if mode == "dev" && env == "production" {
		return errors.New("dev authentication is forbidden in production")
	}
	if mode != "dev" && mode != "cloudflare" {
		return errors.New("AUTH_MODE must be dev or cloudflare")
	}
	return nil
}

type Verifier struct {
	Issuer, Audience, Email string
	Keys                    map[string]*rsa.PublicKey
	mu                      sync.Mutex
	fetched                 time.Time
	Client                  *http.Client
}

func (v *Verifier) Verify(raw string) error {
	if raw == "" {
		return errors.New("missing token")
	}
	claims := jwt.MapClaims{}
	token, e := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) { kid, _ := t.Header["kid"].(string); return v.key(kid) }, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(v.Issuer), jwt.WithAudience(v.Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if e != nil || !token.Valid {
		return errors.New("invalid token")
	}
	email, _ := claims["email"].(string)
	if email == "" || !strings.EqualFold(email, v.Email) {
		return errors.New("wrong owner")
	}
	return nil
}
func (v *Verifier) key(kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if key := v.Keys[kid]; key != nil && (v.fetched.IsZero() || time.Since(v.fetched) < time.Hour) {
		return key, nil
	}
	if time.Since(v.fetched) < time.Minute {
		return nil, errors.New("unknown key")
	}
	v.fetched = time.Now()
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	res, e := client.Get(v.Issuer + "/cdn-cgi/access/certs")
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New("keys unavailable")
	}
	var jwks struct{ Keys []struct{ Kid, N, E string } }
	if e = json.NewDecoder(http.MaxBytesReader(nil, res.Body, 1<<20)).Decode(&jwks); e != nil {
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
	if k := keys[kid]; k != nil {
		return k, nil
	}
	return nil, errors.New("unknown signing key")
}
