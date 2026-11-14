package domain

import (
	"strings"
	"unicode/utf8"
)

func validateDetails(i *Item) error {
	details := make([]Detail, 0, len(i.Details))
	for _, d := range i.Details {
		d.Label, d.Value = strings.TrimSpace(d.Label), strings.TrimSpace(d.Value)
		if d.Label == "" || d.Value == "" {
			continue
		}
		if utf8.RuneCountInString(d.Label) > 60 || utf8.RuneCountInString(d.Value) > 2048 {
			return &Invalid{"details", "Use detail labels up to 60 characters and values up to 2,048 characters."}
		}
		details = append(details, d)
	}
	if len(details) > 10 {
		return &Invalid{"details", "Use at most 10 details."}
	}
	i.Details = details
	return nil
}

// DetailsText preserves the order and literal values in a change preview.
func DetailsText(details []Detail) string {
	lines := make([]string, 0, len(details))
	for _, d := range details {
		lines = append(lines, d.Label+": "+d.Value)
	}
	return strings.Join(lines, "\n")
}
