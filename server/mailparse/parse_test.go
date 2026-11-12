package mailparse

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

func TestBinaryAttachmentAndHTML(t *testing.T) {
	raw := "From: owner@example.com\r\nSubject: =?UTF-8?Q?Booking_confirmation?=\r\nContent-Type: multipart/mixed; boundary=x\r\n\r\n--x\r\nContent-Type: text/html\r\n\r\n<p>Hotel booking</p><script>steal()</script><img src='https://example.invalid/pixel'>\r\n--x\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=booking.bin\r\nContent-Transfer-Encoding: base64\r\n\r\n" + base64.StdEncoding.EncodeToString([]byte{0, 255, 42}) + "\r\n--x--\r\n"
	d, e := (&Parser{}).Parse(context.Background(), strings.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	if d.Subject != "Booking confirmation" || len(d.Attachments) != 1 {
		t.Fatalf("%+v", d)
	}
	if !strings.Contains(d.Parts[0].Text, "Hotel booking") || strings.Contains(d.Parts[0].Text, "steal") {
		t.Fatal(d.Parts[0].Text)
	}
	var found bool
	for _, p := range d.Parts {
		if p.Name == "booking.bin" {
			found = bytes.Equal(p.Data, []byte{0, 255, 42})
		}
	}
	if !found {
		t.Fatal("binary attachment changed")
	}
}
func TestCalendarAndUnsupportedCoverage(t *testing.T) {
	raw := "From: owner@example.com\r\nContent-Type: text/calendar\r\n\r\nBEGIN:VCALENDAR\r\nVERSION:2.0\r\nMETHOD:CANCEL\r\nBEGIN:VEVENT\r\nUID:booking-1\r\nSEQUENCE:2\r\nDTSTART;TZID=Europe/Amsterdam:20261112T090000\r\nSUMMARY:Appointment\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	d, e := (&Parser{}).Parse(context.Background(), strings.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	if len(d.Parts) == 0 || !strings.Contains(d.Parts[0].Text, "booking-1") || !strings.Contains(d.Parts[0].Text, "CANCEL") {
		t.Fatal(d)
	}
}
