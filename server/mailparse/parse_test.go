package mailparse

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
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

func TestDeclaredTextCharset(t *testing.T) {
	raw := append([]byte("From: owner@example.com\r\nContent-Type: text/plain; charset=iso-8859-1\r\n\r\nCaf"), 0xe9)
	document, e := (&Parser{}).Parse(context.Background(), bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	if document.Parts[0].Text != "Café" {
		t.Fatal("declared charset was not decoded")
	}
}

func testPDF(pages int) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	object := func(id int, body string) {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", id, body)
	}
	object(1, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := ""
	for n := 0; n < pages; n++ {
		kids += fmt.Sprintf("%d 0 R ", n+3)
	}
	object(2, fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", pages, kids))
	font, content := pages+3, pages+4
	for n := 0; n < pages; n++ {
		object(n+3, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 100] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", font, content))
	}
	object(font, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	stream := "BT /F1 14 Tf 20 60 Td (Booking ABC123) Tj ET"
	object(content, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return b.Bytes()
}
func TestPDFTextAndPageLimit(t *testing.T) {
	if _, e := exec.LookPath("pdfinfo"); e != nil {
		t.Skip("Poppler runtime check runs separately")
	}
	for _, pages := range []int{1, 51} {
		raw := append([]byte("From: owner@example.com\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=booking.pdf\r\n\r\n"), testPDF(pages)...)
		document, e := (&Parser{}).Parse(context.Background(), bytes.NewReader(raw))
		if e != nil {
			t.Fatal(e)
		}
		found := false
		for _, part := range document.Parts {
			if strings.Contains(part.Text, "Booking ABC123") {
				found = true
			}
		}
		if !found {
			t.Fatal("PDF text missing", document.Coverage)
		}
		if pages > 50 && document.Coverage[0].Status == "parsed" {
			t.Fatal("page truncation reported complete")
		}
	}
}

func TestHTMLBookingContactTargetsAndReferenceSurviveConversion(t *testing.T) {
	raw := "From: owner@example.com\r\nContent-Type: text/html; charset=utf-8\r\n\r\n" +
		`<table><tr><td>Booking reference:</td><td>ABC123456789</td></tr></table><p><a href="tel:+31201234567">Call us</a> <a href="mailto:desk@example.com">Email reception</a> <a href="https://example.com/booking?ref=ABC123456789">Manage booking</a></p><script>private-script</script><a href="javascript:bad()">Bad link</a>`
	document, err := (&Parser{}).Parse(context.Background(), strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	text := document.Parts[0].Text
	for _, want := range []string{"Booking reference:", "ABC123456789", "tel:+31201234567", "mailto:desk@example.com", "https://example.com/booking?ref=ABC123456789"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
	if strings.Contains(text, "reference:ABC") {
		t.Error("table cells ran together")
	}
	if strings.Contains(text, "javascript:") || strings.Contains(text, "private-script") {
		t.Error("active content entered converted text")
	}
}
