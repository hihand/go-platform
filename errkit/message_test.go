package errkit_test

import (
	"errors"
	"testing"

	"github.com/hihand/go-platform/errkit"
)

func TestMessageOf(t *testing.T) {
	t.Parallel()
	if got := errkit.MessageOf(nil); got != "" {
		t.Errorf("MessageOf(nil): want empty, got %q", got)
	}
	if got := errkit.MessageOf(errors.New("plain")); got != "" {
		t.Errorf("MessageOf(non-errkit): want empty, got %q", got)
	}
	if got := errkit.MessageOf(errkit.NotFound("user 42")); got != "user 42" {
		t.Errorf("MessageOf(direct): want %q, got %q", "user 42", got)
	}
}
