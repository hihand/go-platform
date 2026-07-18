package errkit_test

import (
	"errors"
	"io"
	"testing"

	"github.com/hihand/go-platform/errkit"
)

func TestNew_Defaults(t *testing.T) {
	t.Parallel()
	err := errkit.New()
	if err == nil {
		t.Fatalf("New() returned nil")
	}
	if got := err.Code(); got != errkit.CodeUnknown {
		t.Errorf("default code: want %q, got %q", errkit.CodeUnknown, got)
	}
	if got := err.Message(); got != "" {
		t.Errorf("default message: want empty, got %q", got)
	}
	if got := err.Error(); got != "UNKNOWN: " {
		t.Errorf("default Error(): want %q, got %q", "UNKNOWN: ", got)
	}
	if err.Unwrap() != nil {
		t.Errorf("default cause: want nil, got %v", err.Unwrap())
	}
}

func TestNew_WithCodeAndMessage(t *testing.T) {
	t.Parallel()
	err := errkit.New(
		errkit.WithCode(errkit.CodeNotFound),
		errkit.WithMessage("user 42"),
	)
	if err.Code() != errkit.CodeNotFound {
		t.Errorf("code: want %q, got %q", errkit.CodeNotFound, err.Code())
	}
	if err.Message() != "user 42" {
		t.Errorf("message: want %q, got %q", "user 42", err.Message())
	}
	if got, want := err.Error(), "NOT_FOUND: user 42"; got != want {
		t.Errorf("Error(): want %q, got %q", want, got)
	}
}

func TestWrap_NilReturnsNil(t *testing.T) {
	t.Parallel()
	if err := errkit.Wrap(nil); err != nil {
		t.Errorf("Wrap(nil) must return nil, got %v", err)
	}
}

func TestWrap_PreservesCauseAndIs(t *testing.T) {
	t.Parallel()
	root := io.EOF
	err := errkit.Wrap(root,
		errkit.WithCode(errkit.CodeInternal),
		errkit.WithMessage("wrapped"),
	)
	if !errors.Is(err, io.EOF) {
		t.Errorf("errors.Is should match io.EOF in chain")
	}
	if got, want := err.Error(), "INTERNAL: wrapped: EOF"; got != want {
		t.Errorf("Error(): want %q, got %q", want, got)
	}
}

func TestSugarConstructors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  errkit.Error
		code errkit.Code
	}{
		{"NotFound", errkit.NotFound("nf"), errkit.CodeNotFound},
		{"InvalidArgument", errkit.InvalidArgument("ia"), errkit.CodeInvalidArgument},
		{"Internal", errkit.Internal("in"), errkit.CodeInternal},
		{"AlreadyExists", errkit.AlreadyExists("ae"), errkit.CodeAlreadyExists},
		{"Unauthenticated", errkit.Unauthenticated("ua"), errkit.CodeUnauthenticated},
		{"PermissionDenied", errkit.PermissionDenied("pd"), errkit.CodePermissionDenied},
		{"Unavailable", errkit.Unavailable("uv"), errkit.CodeUnavailable},
		{"DeadlineExceeded", errkit.DeadlineExceeded("de"), errkit.CodeDeadlineExceeded},
		{"Canceled", errkit.Canceled("cx"), errkit.CodeCanceled},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got.Code() != tc.code {
				t.Errorf("%s: code want %q, got %q", tc.name, tc.code, tc.got.Code())
			}
		})
	}
}
