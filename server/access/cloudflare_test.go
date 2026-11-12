package access

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTRejectsInvalidIdentity(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &cloudflareVerifier{
		Issuer: "https://team.cloudflareaccess.com", Audience: "aud", Email: "owner@example.com",
		Keys: map[string]*rsa.PublicKey{"key": &key.PublicKey},
	}
	for _, tc := range []struct {
		name, issuer, email, audience, header string
		expiry                                time.Time
		key                                   *rsa.PrivateKey
		valid                                 bool
	}{
		{"valid", verifier.Issuer, verifier.Email, "aud", "Cf-Access-Jwt-Assertion", time.Now().Add(time.Hour), key, true},
		{"signature", verifier.Issuer, verifier.Email, "aud", "Cf-Access-Jwt-Assertion", time.Now().Add(time.Hour), other, false},
		{"expired", verifier.Issuer, verifier.Email, "aud", "Cf-Access-Jwt-Assertion", time.Now().Add(-time.Hour), key, false},
		{"missing expiry", verifier.Issuer, verifier.Email, "aud", "Cf-Access-Jwt-Assertion", time.Time{}, key, false},
		{"issuer", "https://other.example", verifier.Email, "aud", "Cf-Access-Jwt-Assertion", time.Now().Add(time.Hour), key, false},
		{"audience", verifier.Issuer, verifier.Email, "wrong", "Cf-Access-Jwt-Assertion", time.Now().Add(time.Hour), key, false},
		{"owner", verifier.Issuer, "wrong@example.com", "aud", "Cf-Access-Jwt-Assertion", time.Now().Add(time.Hour), key, false},
		{"wrong header", verifier.Issuer, verifier.Email, "aud", "Authorization", time.Now().Add(time.Hour), key, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": tc.issuer, "aud": []string{tc.audience}, "email": tc.email}
			if !tc.expiry.IsZero() {
				claims["exp"] = tc.expiry.Unix()
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "key"
			signed, err := token.SignedString(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("GET", "/v1/chat", nil)
			request.Header.Set(tc.header, signed)
			if err := verifier.Authenticate(request); (err == nil) != tc.valid {
				t.Fatalf("want valid %v, got %v", tc.valid, err)
			}
		})
	}
}
