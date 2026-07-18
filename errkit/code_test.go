package errkit_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/hihand/go-platform/errkit"
)

func TestCodeOf_NilAndUnknown(t *testing.T) {
	t.Parallel()
	if got := errkit.CodeOf(nil); got != errkit.CodeUnknown {
		t.Errorf("CodeOf(nil): want %q, got %q", errkit.CodeUnknown, got)
	}
	if got := errkit.CodeOf(errors.New("plain")); got != errkit.CodeUnknown {
		t.Errorf("CodeOf(non-errkit): want %q, got %q", errkit.CodeUnknown, got)
	}
}

func TestCodeOf_Wrapping(t *testing.T) {
	t.Parallel()
	base := errkit.NotFound("x")
	wrapped := fmt.Errorf("ctx: %w", base)
	if got := errkit.CodeOf(wrapped); got != errkit.CodeNotFound {
		t.Errorf("CodeOf across plain wrap: want %q, got %q", errkit.CodeNotFound, got)
	}
}

func TestFromError_BoolMatrix(t *testing.T) {
	t.Parallel()
	if _, ok := errkit.FromError(nil); ok {
		t.Errorf("FromError(nil): want ok=false")
	}
	if _, ok := errkit.FromError(errors.New("x")); ok {
		t.Errorf("FromError(non-errkit): want ok=false")
	}
	e, ok := errkit.FromError(errkit.Internal("db"))
	if !ok {
		t.Fatalf("FromError(errkit): want ok=true")
	}
	if e.Code() != errkit.CodeInternal {
		t.Errorf("FromError returned wrong code: %q", e.Code())
	}
}

func TestIsCode(t *testing.T) {
	t.Parallel()
	if errkit.IsCode(nil, errkit.CodeNotFound) {
		t.Errorf("IsCode(nil) should be false")
	}
	if !errkit.IsCode(errkit.NotFound(""), errkit.CodeNotFound) {
		t.Errorf("direct match failed")
	}
	// walks chain
	wrapped := fmt.Errorf("outer: %w", errkit.NotFound("inner"))
	if !errkit.IsCode(wrapped, errkit.CodeNotFound) {
		t.Errorf("chain match failed")
	}
	// wrong code
	if errkit.IsCode(errkit.NotFound(""), errkit.CodeInternal) {
		t.Errorf("wrong-code match should be false")
	}
}

func TestErrorsAs_PointerToInterface(t *testing.T) {
	t.Parallel()
	var target errkit.Error
	if !errors.As(errkit.Wrap(io.ErrClosedPipe, errkit.WithCode(errkit.CodeInternal)), &target) {
		t.Errorf("errors.As did not match errkit.Error in chain")
	}
	if target.Code() != errkit.CodeInternal {
		t.Errorf("matched wrong code: %q", target.Code())
	}
}
