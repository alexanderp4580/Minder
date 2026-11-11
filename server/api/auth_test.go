package api

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/golang-jwt/jwt/v5"
	"testing"
	"time"
)

func TestJWTRejectsInvalidIdentity(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := &Verifier{Issuer: "https://team.cloudflareaccess.com", Audience: "aud", Email: "owner@example.com", Keys: map[string]*rsa.PublicKey{"key": &key.PublicKey}}
	for _, tc := range []struct {
		name, email, aud string
		expiry           time.Time
		key              *rsa.PrivateKey
		ok               bool
	}{{"valid", "owner@example.com", "aud", time.Now().Add(time.Hour), key, true}, {"signature", "owner@example.com", "aud", time.Now().Add(time.Hour), other, false}, {"expired", "owner@example.com", "aud", time.Now().Add(-time.Hour), key, false}, {"audience", "owner@example.com", "wrong", time.Now().Add(time.Hour), key, false}, {"owner", "wrong@example.com", "aud", time.Now().Add(time.Hour), key, false}} {
		t.Run(tc.name, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": v.Issuer, "aud": []string{tc.aud}, "email": tc.email, "exp": tc.expiry.Unix()}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "key"
			signed, _ := token.SignedString(tc.key)
			if e := v.Verify(signed); (e == nil) != tc.ok {
				t.Fatalf("valid %v: %v", tc.ok, e)
			}
		})
	}
}
