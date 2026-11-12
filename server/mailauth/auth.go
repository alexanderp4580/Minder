// Package mailauth authenticates a dedicated email Worker; it grants no owner privileges.
package mailauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/alexanderp4580/Minder/server/config"
	"github.com/alexanderp4580/Minder/server/domain"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

type Verifier struct{ settings config.EmailSettings }

func FromEnvironment(getenv func(string) string) (*Verifier, error) {
	cfg, e := config.LoadEmail(getenv)
	if e != nil {
		return nil, e
	}
	if !cfg.Enabled {
		return nil, nil
	}
	return &Verifier{cfg}, nil
}
func (v *Verifier) Authenticate(r *http.Request, raw []byte, now time.Time) (out domain.CaptureEnvelope, err error) {
	deny := func(reason string) error { return &Denied{reason} }
	if v == nil || len(raw) > domain.MaxEmailBytes {
		return out, deny("unavailable_or_too_large")
	}
	stamp := r.Header.Get("X-Minder-Email-Time")
	ts, e := strconv.ParseInt(stamp, 10, 64)
	if e != nil || ts < now.Add(-5*time.Minute).Unix() || ts > now.Add(5*time.Minute).Unix() {
		return out, deny("bad_timestamp")
	}
	id := r.Header.Get("X-Minder-Email-ID")
	if len(id) < 1 || len(id) > 128 || strings.ContainsAny(id, "\r\n") {
		return out, deny("bad_id")
	}
	meta := r.Header.Get("X-Minder-Email-Envelope")
	if len(meta) > 2048 {
		return out, deny("envelope_too_large")
	}
	digest := sha256.Sum256(raw)
	payload := "v1\n" + stamp + "\n" + id + "\n" + meta + "\n" + hex.EncodeToString(digest[:])
	h := hmac.New(sha256.New, v.settings.HMACKey)
	h.Write([]byte(payload))
	signature, e := hex.DecodeString(r.Header.Get("X-Minder-Email-Signature"))
	if e != nil || !hmac.Equal(signature, h.Sum(nil)) {
		return out, deny("bad_signature")
	}
	data, e := base64.RawURLEncoding.DecodeString(meta)
	if e != nil {
		return out, deny("bad_envelope_encoding")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if dec.Decode(&out) != nil {
		return domain.CaptureEnvelope{}, deny("bad_envelope_fields")
	}
	if out.DMARC != "pass" {
		return domain.CaptureEnvelope{}, deny("dmarc_not_pass")
	}
	if !strings.EqualFold(out.To, v.settings.Recipient) {
		return domain.CaptureEnvelope{}, deny("recipient_mismatch")
	}
	allowed := false
	for _, a := range v.settings.AllowedSenders {
		if strings.EqualFold(a, out.From) {
			allowed = true
		}
	}
	if !allowed {
		return domain.CaptureEnvelope{}, deny("sender_not_allowed")
	}
	msg, e := mail.ReadMessage(bytes.NewReader(raw))
	if e != nil || len(msg.Header["From"]) != 1 {
		return domain.CaptureEnvelope{}, deny("bad_message_from")
	}
	from, e := mail.ParseAddressList(msg.Header.Get("From"))
	if e != nil || len(from) != 1 || !strings.EqualFold(from[0].Address, out.From) {
		return domain.CaptureEnvelope{}, deny("from_header_mismatch")
	}
	out.RequestID = id
	return out, nil
}

// Denied hides which check failed from the caller; Reason feeds the server log only.
type Denied struct{ reason string }

func (d *Denied) Error() string  { return "Email sender or signature is invalid" }
func (d *Denied) Reason() string { return d.reason }
