package mailauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSignedSenderAndBody(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	env := map[string]string{"EMAIL_CAPTURE_ENABLED": "true", "EMAIL_ALLOWED_SENDERS": "Owner@example.com", "EMAIL_RECIPIENT": "assistant@example.net", "EMAIL_INGEST_HMAC_KEY": base64.StdEncoding.EncodeToString(key), "EMAIL_SOURCE_DIR": t.TempDir()}
	v, e := FromEnvironment(func(k string) string { return env[k] })
	if e != nil {
		t.Fatal(e)
	}
	raw := []byte("From: Owner <owner@example.com>\r\nTo: assistant@example.net\r\n\r\nBooking")
	req := httptest.NewRequest("POST", "/v1/email/ingest", bytes.NewReader(raw))
	meta := base64.RawURLEncoding.EncodeToString([]byte(`{"from":"owner@example.com","to":"assistant@example.net","dmarc":"pass"}`))
	req.Header.Set("X-Minder-Email-Time", "1794528000")
	req.Header.Set("X-Minder-Email-ID", "request1")
	req.Header.Set("X-Minder-Email-Envelope", meta)
	digest := sha256.Sum256(raw)
	payload := "v1\n1794528000\nrequest1\n" + meta + "\n" + hex.EncodeToString(digest[:])
	h := hmac.New(sha256.New, key)
	h.Write([]byte(payload))
	req.Header.Set("X-Minder-Email-Signature", hex.EncodeToString(h.Sum(nil)))
	now := time.Unix(1794528000, 0)
	if _, e = v.Authenticate(req, raw, now); e != nil {
		t.Fatal(e)
	}
	if _, e = v.Authenticate(req, append(raw, '!'), now); e == nil {
		t.Fatal("tampered body accepted")
	}
	if _, e = v.Authenticate(req, raw, now.Add(6*time.Minute)); e == nil {
		t.Fatal("expired accepted")
	}
	req.Header.Set("X-Minder-Email-Envelope", strings.ReplaceAll(meta, "x", "y"))
	if _, e = v.Authenticate(req, raw, now); e == nil {
		t.Fatal("tampered metadata accepted")
	}
}
func TestMissingAllowlistClosed(t *testing.T) {
	if _, e := FromEnvironment(func(k string) string {
		if k == "EMAIL_CAPTURE_ENABLED" {
			return "true"
		}
		return ""
	}); e == nil {
		t.Fatal("missing config accepted")
	}
	v, e := FromEnvironment(func(string) string { return "" })
	if e != nil || v != nil {
		t.Fatal("legacy startup changed", e)
	}
}

func TestWorkerSignatureFixture(t *testing.T) {
	raw, e := os.ReadFile("../../toolchain/email-ingress-vectors.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixture struct {
		Raw       string
		Timestamp int64
		KeyBase64 string
		Headers   map[string]string
	}
	if e = json.Unmarshal(raw, &fixture); e != nil {
		t.Fatal(e)
	}
	values := map[string]string{"EMAIL_CAPTURE_ENABLED": "true", "EMAIL_ALLOWED_SENDERS": "owner@example.com", "EMAIL_RECIPIENT": "assistant@example.net", "EMAIL_INGEST_HMAC_KEY": fixture.KeyBase64, "EMAIL_SOURCE_DIR": t.TempDir()}
	verifier, e := FromEnvironment(func(k string) string { return values[k] })
	if e != nil {
		t.Fatal(e)
	}
	request := httptest.NewRequest("POST", "/v1/email/ingest", strings.NewReader(fixture.Raw))
	for k, v := range fixture.Headers {
		request.Header.Set(k, v)
	}
	envelope, e := verifier.Authenticate(request, []byte(fixture.Raw), time.Unix(fixture.Timestamp, 0))
	if e != nil || envelope.RequestID != "request1" {
		t.Fatal(envelope, e)
	}
}
