package domain

import "time"

func validateEvent(i *Item) error {
	if i.Kind == "" {
		i.Kind = "task"
	}
	if i.Kind != "task" && i.Kind != "event" && i.Kind != "reminder" {
		return &Invalid{"kind", "Choose task, event or reminder."}
	}
	for _, z := range []string{i.StartTimezone, i.EndTimezone} {
		if z != "" {
			if _, e := time.LoadLocation(z); e != nil {
				return &Invalid{"timezone", "Choose a valid timezone."}
			}
		}
	}
	for _, f := range []struct{ name, date, instant string }{{"start_date", i.StartDate, i.StartAt}, {"end_date", i.EndDate, i.EndAt}, {"due_date", i.DueDate, i.DueAt}} {
		if f.date == "" {
			continue
		}
		if _, e := time.Parse(time.DateOnly, f.date); e != nil || f.instant != "" {
			return &Invalid{f.name, "Use a valid date or a timestamp, not both."}
		}
	}
	if i.EndDate != "" && (i.StartDate == "" || i.EndDate <= i.StartDate) {
		return &Invalid{"end_date", "Set an end date after the start date."}
	}
	if i.EndAt != "" {
		end, e := time.Parse(time.RFC3339, i.EndAt)
		start, se := time.Parse(time.RFC3339, i.StartAt)
		if e != nil || se != nil || !end.After(start) {
			return &Invalid{"end_at", "Set an end after the start time."}
		}
		minutes := int(end.Sub(start) / time.Minute)
		if i.DurationMinutes != 0 && i.DurationMinutes != minutes {
			return &Invalid{"duration_minutes", "Duration must match the start and end."}
		}
		i.EndAt = end.UTC().Format(time.RFC3339)
		i.DurationMinutes = minutes
	}
	if i.StartDate != "" && i.DueDate != "" && i.StartDate > i.DueDate {
		return &Invalid{"start_date", "The start must not follow the deadline."}
	}
	return nil
}

// Overlaps uses a half-open range. Date-only events follow the viewer's calendar dates.
func Overlaps(i Item, from, to time.Time, zone *time.Location) bool {
	if zone == nil {
		zone = time.UTC
	}
	if i.StartDate != "" || i.DueDate != "" {
		date := i.StartDate
		if date == "" {
			date = i.DueDate
		}
		start, e := time.ParseInLocation(time.DateOnly, date, zone)
		if e != nil {
			return false
		}
		end := start.AddDate(0, 0, 1)
		if i.StartDate != "" && i.EndDate != "" {
			end, e = time.ParseInLocation(time.DateOnly, i.EndDate, zone)
			if e != nil {
				return false
			}
		}
		return start.Before(to) && end.After(from)
	}
	start, e := time.Parse(time.RFC3339, PlacedAt(i))
	if e != nil {
		return false
	}
	if i.EndAt != "" {
		end, e := time.Parse(time.RFC3339, i.EndAt)
		return e == nil && start.Before(to) && end.After(from)
	}
	return !start.Before(from) && start.Before(to)
}
