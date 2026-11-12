package mailparse

import (
	"bytes"
	"github.com/arran4/golang-ical"
)

func calendarText(data []byte) (string, error) {
	cal, e := ics.ParseCalendar(bytes.NewReader(data))
	if e != nil {
		return "", e
	}
	return cal.Serialize(), nil
}
