package domain

import (
	"errors"
	"fmt"
)

var ErrBusy = errors.New("Wait for Minder to finish.")

func ErrorCode(e error) string {
	var invalid *Invalid
	var model *ModelError
	switch {
	case errors.Is(e, ErrBusy):
		return "busy"
	case errors.As(e, &invalid):
		return "invalid_field"
	case errors.Is(e, ErrConflict):
		return "conflict"
	case errors.Is(e, ErrNotFound):
		return "not_found"
	case errors.Is(e, ErrLimit):
		return "limit_reached"
	case errors.As(e, &model):
		return "model_error"
	default:
		return "internal_error"
	}
}

// Describe gives the owner a short message. Model and internal failures show a fixed text, never their details.
func Describe(e error) string {
	switch ErrorCode(e) {
	case "model_error", "internal_error":
		return "Minder could not finish. Try again."
	}
	return fmt.Sprint(e)
}
