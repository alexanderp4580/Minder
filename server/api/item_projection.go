package api

import (
	"github.com/alexanderp4580/Minder/server/domain"
	"time"
)

type calendarItem struct {
	domain.Item
	StartLocal string `json:"start_local,omitempty"`
	EndLocal   string `json:"end_local,omitempty"`
}

func localTimeLabel(instant, zone string) string {
	if instant == "" || zone == "" {
		return ""
	}
	at, e := time.Parse(time.RFC3339, instant)
	if e != nil {
		return ""
	}
	loc, e := time.LoadLocation(zone)
	if e != nil {
		return ""
	}
	return at.In(loc).Format("2006-01-02T15:04")
}
func calendarResponse(items []domain.Item) []calendarItem {
	out := make([]calendarItem, 0, len(items))
	for _, item := range items {
		out = append(out, calendarItem{Item: item, StartLocal: localTimeLabel(item.StartAt, item.StartTimezone), EndLocal: localTimeLabel(item.EndAt, item.EndTimezone)})
	}
	return out
}
