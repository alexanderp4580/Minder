package domain

import (
	"context"
	"errors"
)

// FailureReason says why an email could not be read. Each reason has one fixed sentence, so the
// owner never sees model text or internal detail.
type FailureReason string

const (
	ReasonCutOff            FailureReason = "cut-off"
	ReasonEvidence          FailureReason = "evidence"
	ReasonNoReview          FailureReason = "no-review"
	ReasonUnreadable        FailureReason = "unreadable"
	ReasonUnparsed          FailureReason = "unparsed"
	ReasonTimeout           FailureReason = "timeout"
	ReasonLimit             FailureReason = "limit"
	ReasonOther             FailureReason = "other"
	ReasonReviewItemMissing FailureReason = "review_item_missing"
)

var failureText = map[FailureReason]string{
	ReasonCutOff:            "the model's answer was cut off.",
	ReasonEvidence:          "the details did not match the email text.",
	ReasonNoReview:          "the model returned no review.",
	ReasonUnreadable:        "the model's answer was unreadable.",
	ReasonUnparsed:          "the email could not be parsed.",
	ReasonTimeout:           "the model took too long.",
	ReasonLimit:             "the monthly model limit is reached.",
	ReasonOther:             "something went wrong.",
	ReasonReviewItemMissing: "the model did not return the item being reviewed.",
}

// Sentence is the message the owner sees in the email entry.
func (r FailureReason) Sentence() string {
	text, ok := failureText[r]
	if !ok {
		text = failureText[ReasonOther]
	}
	return "I could not read this email: " + text + " The original is kept."
}

// ReasonOf classifies an extraction error.
func ReasonOf(err error) FailureReason {
	var model *ModelError
	switch {
	case errors.Is(err, ErrLimit):
		return ReasonLimit
	case errors.Is(err, context.DeadlineExceeded):
		return ReasonTimeout
	case errors.As(err, &model) && model.Reason != "":
		return model.Reason
	}
	return ReasonOther
}
