package errkit_test

import (
	"errors"
	"testing"

	"github.com/hihand/go-platform/errkit"
)

func TestWithCause_PropagatesUnwrap(t *testing.T) {
	t.Parallel()
	root := errors.New("root")
	err := errkit.New(
		errkit.WithCause(root),
		errkit.WithCode(errkit.CodeInternal),
		errkit.WithMessage("boom"),
	)
	if got := err.Unwrap(); got != root {
		t.Errorf("Unwrap: want %v, got %v", root, got)
	}
	if !errors.Is(err, root) {
		t.Errorf("errors.Is should match root cause")
	}
}

func TestWithCode_Overrides(t *testing.T) {
	t.Parallel()
	// last one wins
	err := errkit.New(
		errkit.WithCode(errkit.CodeInternal),
		errkit.WithCode(errkit.CodeNotFound),
	)
	if err.Code() != errkit.CodeNotFound {
		t.Errorf("last WithCode should win: got %q", err.Code())
	}
}

func TestWithMessage_Overrides(t *testing.T) {
	t.Parallel()
	err := errkit.New(
		errkit.WithMessage("first"),
		errkit.WithMessage("second"),
	)
	if err.Message() != "second" {
		t.Errorf("last WithMessage should win: got %q", err.Message())
	}
}
