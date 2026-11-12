package config

import (
	"encoding/base64"
	"errors"
	"net/mail"
	"strconv"
	"strings"
)

type EmailSettings struct {
	Enabled              bool
	AllowedSenders       []string
	Recipient, SourceDir string
	HMACKey              []byte
}

func LoadEmail(getenv func(string) string) (out EmailSettings, err error) {
	raw := getenv("EMAIL_CAPTURE_ENABLED")
	if raw == "" {
		return out, nil
	}
	out.Enabled, err = strconv.ParseBool(raw)
	if err != nil {
		return out, errors.New("EMAIL_CAPTURE_ENABLED must be true or false")
	}
	if !out.Enabled {
		return out, nil
	}
	invalid := func(key string) (EmailSettings, error) {
		return EmailSettings{}, errors.New(key + " is missing or invalid")
	}
	for _, v := range strings.Split(getenv("EMAIL_ALLOWED_SENDERS"), ",") {
		v = strings.TrimSpace(v)
		a, e := mail.ParseAddress(v)
		if e != nil || a.Address != v || !strings.Contains(v, "@") || a.Name != "" {
			return invalid("EMAIL_ALLOWED_SENDERS")
		}
		out.AllowedSenders = append(out.AllowedSenders, strings.ToLower(v))
	}
	out.Recipient = getenv("EMAIL_RECIPIENT")
	a, e := mail.ParseAddress(out.Recipient)
	if e != nil || a.Address != out.Recipient || a.Name != "" {
		return invalid("EMAIL_RECIPIENT")
	}
	out.Recipient = strings.ToLower(out.Recipient)
	out.HMACKey, e = base64.StdEncoding.DecodeString(getenv("EMAIL_INGEST_HMAC_KEY"))
	if e != nil || len(out.HMACKey) < 32 {
		return invalid("EMAIL_INGEST_HMAC_KEY")
	}
	out.SourceDir = getenv("EMAIL_SOURCE_DIR")
	if out.SourceDir == "" {
		return invalid("EMAIL_SOURCE_DIR")
	}
	return out, nil
}
