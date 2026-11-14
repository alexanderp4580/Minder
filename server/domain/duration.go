package domain

import "strings"

// AsksDuration reports whether a question is about how long something takes.
func AsksDuration(question string) bool {
	question = strings.ToLower(question)
	return strings.Contains(question, "how long") || strings.Contains(question, "duration")
}
