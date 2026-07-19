package errkit_test

import (
	"errors"
	"fmt"
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

func TestMetadataOf_DefaultsToEmpty(t *testing.T) {
	t.Parallel()
	if got := errkit.MetadataOf(nil); len(got) != 0 {
		t.Errorf("MetadataOf(nil) want empty map, got %v", got)
	}
	if got := errkit.MetadataOf(errors.New("plain")); len(got) != 0 {
		t.Errorf("MetadataOf(non-errkit) want empty map, got %v", got)
	}
	if got := errkit.MetadataOf(errkit.New()); got == nil {
		t.Errorf("MetadataOf(no metadata) want non-nil empty map")
	}
}

func TestMetadata_DefensiveCopy(t *testing.T) {
	t.Parallel()
	err := errkit.New(errkit.WithMetadata(map[string]any{
		"request_id": "abc-123",
		"retries":    3,
	}))
	md := errkit.MetadataOf(err)
	if md["request_id"] != "abc-123" {
		t.Errorf("metadata not present")
	}
	// mutate the returned map; original must not change.
	md["request_id"] = "tampered"
	if got := errkit.MetadataOf(err)["request_id"]; got != "abc-123" {
		t.Errorf("WithMetadata did not defensive-copy on construction; got %v", got)
	}
}

func TestMetadata_DefensiveCopy_OnWithMetadata(t *testing.T) {
	t.Parallel()
	src := map[string]any{"k": "v"}
	err := errkit.New(errkit.WithMetadata(src))
	// mutate after construction; must not leak into the error.
	src["k"] = "tampered"
	if got := errkit.MetadataOf(err)["k"]; got != "v" {
		t.Errorf("WithMetadata did not defensive-copy on input; got %v", got)
	}
}

func TestMetadata_EmptyMapIsNoOp(t *testing.T) {
	t.Parallel()
	err := errkit.New(errkit.WithMetadata(map[string]any{}))
	if got := errkit.MetadataOf(err); len(got) != 0 {
		t.Errorf("empty WithMetadata should leave metadata empty, got %v", got)
	}
}

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

func TestWithStack_NonPublicEntry(t *testing.T) {
	t.Parallel()
	a := errkit.New()
	b := errkit.New(errkit.WithStack())
	if a.Code() != b.Code() || a.Message() != b.Message() {
		t.Errorf("WithStack should not change Code/Message: a=%v b=%v", a, b)
	}
	if a.Error() != b.Error() {
		t.Errorf("WithStack should not change Error(): a=%q b=%q", a.Error(), b.Error())
	}
}