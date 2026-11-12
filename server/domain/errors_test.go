package domain_test

import (
	"errors"
	"testing"

	"github.com/alexanderp4580/Minder/server/domain"
)

func TestErrorCodesNameTheRealCause(t *testing.T) {
	for _, c := range []struct {
		err  error
		code string
	}{
		{&domain.Invalid{Field: "title", Message: "x"}, "invalid_field"},
		{domain.ErrConflict, "conflict"},
		{domain.ErrNotFound, "not_found"},
		{domain.ErrLimit, "limit_reached"},
		{domain.ErrBusy, "busy"},
		{&domain.ModelError{Message: "The model could not connect."}, "model_error"},
		{errors.New("database is locked"), "internal_error"},
	} {
		if got := domain.ErrorCode(c.err); got != c.code {
			t.Errorf("%v: got %q, want %q", c.err, got, c.code)
		}
	}
	if domain.Describe(errors.New("database is locked")) != "Minder could not finish. Try again." {
		t.Fatal("an internal error must not leak its text")
	}
}
