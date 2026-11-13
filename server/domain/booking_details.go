package domain

import (
	"regexp"
	"strings"
)

var bookingNoteLine = regexp.MustCompile(`^([^:=]+)\s*[:=]\s*(.+)$`)
var bookingNoteSpaced = regexp.MustCompile(`(?i)^(phone|tel\.?|telephone|call|booking reference|booking ref\.?|confirmation number|reservation number)\s+(.+)$`)
var bookingPhone = regexp.MustCompile(`^\+?[0-9() .\t\x{00a0}\x{202f}-]+$`)
var bookingEmail = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Bare Ref/Reference labels stay in notes: older captures use Ref for calendar UID/sequence.
var bookingNoteLabels = map[string]string{
	"phone": "phone", "contact phone": "phone", "phone number": "phone", "telephone": "phone", "tel": "phone", "tel.": "phone", "call": "phone", "telefoon": "phone", "telefoonnummer": "phone",
	"email": "email", "e-mail": "email", "contact email": "email", "email address": "email", "e-mailadres": "email",
	"website": "website", "web": "website", "url": "website",
	"booking reference": "booking_reference", "booking ref": "booking_reference", "booking ref.": "booking_reference", "booking number": "booking_reference", "confirmation number": "booking_reference", "confirmation code": "booking_reference", "reservation number": "booking_reference", "reservation reference": "booking_reference", "boekingsnummer": "booking_reference", "reserveringsnummer": "booking_reference",
}

// NormalizeBookingDetails upgrades labelled older notes without changing unrelated instructions.
// Explicit fields win conflicts; clear requests also remove corresponding legacy note lines.
func NormalizeBookingDetails(item *Item, clear ...string) {
	fields := map[string]*string{"phone": &item.Phone, "email": &item.Email, "website": &item.Website, "booking_reference": &item.BookingReference}
	cleared := map[string]bool{}
	for _, name := range clear {
		cleared[name] = true
	}
	var remaining []string
	for _, raw := range strings.Split(item.Notes, "\n") {
		line := strings.TrimSpace(raw)
		for _, bullet := range []string{"- ", "* ", "• "} {
			if strings.HasPrefix(line, bullet) {
				line = strings.TrimSpace(strings.TrimPrefix(line, bullet))
				break
			}
		}
		match := bookingNoteLine.FindStringSubmatch(line)
		spaced := false
		if match == nil {
			match = bookingNoteSpaced.FindStringSubmatch(line)
			spaced = true
		}
		if match == nil {
			remaining = append(remaining, raw)
			continue
		}
		label, value := strings.ToLower(strings.TrimSpace(match[1])), strings.TrimSpace(match[2])
		key := bookingNoteLabels[label]
		if label == "contact" {
			switch {
			case isBookingPhone(value):
				key = "phone"
			case bookingEmail.MatchString(value):
				key = "email"
			case strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "www."):
				key = "website"
			}
		}
		if spaced && key == "phone" && !isBookingPhone(value) {
			remaining = append(remaining, raw)
			continue
		}
		dst := fields[key]
		if dst == nil {
			remaining = append(remaining, raw)
			continue
		}
		if cleared[key] {
			continue
		}
		if *dst == "" || *dst == value {
			*dst = value
		} else {
			remaining = append(remaining, raw)
		}
	}
	item.Notes = strings.TrimSpace(strings.Join(remaining, "\n"))
	for key := range cleared {
		if dst := fields[key]; dst != nil {
			*dst = ""
		}
	}
}

func isBookingPhone(value string) bool {
	if !bookingPhone.MatchString(value) {
		return false
	}
	digits := 0
	for _, c := range value {
		if c >= '0' && c <= '9' {
			digits++
		}
	}
	return digits >= 7 && digits <= 15
}
