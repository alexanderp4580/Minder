package domain

import (
	"regexp"
	"strconv"
	"strings"
)

var durationPattern = regexp.MustCompile(`^(?:(\d+)\s*(?:h|hr|hrs|hour|hours))?\s*(?:(\d+)\s*(?:m|min|mins|minute|minutes))?$`)

const maxDurationMinutes = 10080

// ParseDuration reads a quick reply such as "45 minutes", "1 hour" or "1 hour 30 minutes" as
// minutes. Anything else, including extra words, is not a duration.
func ParseDuration(text string) (int, bool) {
	text = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(text)), ".")
	match := durationPattern.FindStringSubmatch(text)
	if match == nil || (match[1] == "" && match[2] == "") {
		return 0, false
	}
	hours, _ := strconv.Atoi(match[1])
	minutes, _ := strconv.Atoi(match[2])
	total := hours*60 + minutes
	if total <= 0 || total > maxDurationMinutes {
		return 0, false
	}
	return total, true
}

// AsksDuration reports whether a question is about how long something takes.
func AsksDuration(question string) bool {
	question = strings.ToLower(question)
	return strings.Contains(question, "how long") || strings.Contains(question, "duration")
}
