// Package mailauth authenticates a dedicated email Worker; it grants no owner privileges.
package mailauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	denied := errors.New("Email sender or signature is invalid")
	if v == nil || len(raw) > domain.MaxEmailBytes {
		return out, denied
	}
	stamp := r.Header.Get("X-Minder-Email-Time")
	ts, e := strconv.ParseInt(stamp, 10, 64)
	if e != nil || ts < now.Add(-5*time.Minute).Unix() || ts > now.Add(5*time.Minute).Unix() {
		return out, denied
	}
	id := r.Header.Get("X-Minder-Email-ID")
	if len(id) < 1 || len(id) > 128 || strings.ContainsAny(id, "\r\n") {
		return out, denied
	}
	meta := r.Header.Get("X-Minder-Email-Envelope")
	if len(meta) > 2048 {
		return out, denied
	}
	digest := sha256.Sum256(raw)
	payload := "v1\n" + stamp + "\n" + id + "\n" + meta + "\n" + hex.EncodeToString(digest[:])
	h := hmac.New(sha256.New, v.settings.HMACKey)
	h.Write([]byte(payload))
	signature, e := hex.DecodeString(r.Header.Get("X-Minder-Email-Signature"))
	if e != nil || !hmac.Equal(signature, h.Sum(nil)) {
		return out, denied
	}
	data, e := base64.RawURLEncoding.DecodeString(meta)
	if e != nil {
		return out, denied
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if dec.Decode(&out) != nil || out.DMARC != "pass" || !strings.EqualFold(out.To, v.settings.Recipient) {
		return domain.CaptureEnvelope{}, denied
	}
	allowed := false
	for _, a := range v.settings.AllowedSenders {
		if strings.EqualFold(a, out.From) {
			allowed = true
		}
	}
	if !allowed {
		return domain.CaptureEnvelope{}, denied
	}
	msg, e := mail.ReadMessage(bytes.NewReader(raw))
	if e != nil || len(msg.Header["From"]) != 1 {
		return domain.CaptureEnvelope{}, denied
	}
	from, e := mail.ParseAddressList(msg.Header.Get("From"))
	if e != nil || len(from) != 1 || !strings.EqualFold(from[0].Address, out.From) {
		return domain.CaptureEnvelope{}, denied
	}
	out.RequestID = id
	return out, nil
}
