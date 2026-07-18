package grpcerr_test

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/errkit/grpcerr"
)

func TestToGRPCStatus_DefaultMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code errkit.Code
		want codes.Code
	}{
		{errkit.CodeInvalidArgument, codes.InvalidArgument},
		{errkit.CodeNotFound, codes.NotFound},
		{errkit.CodeAlreadyExists, codes.AlreadyExists},
		{errkit.CodeUnauthenticated, codes.Unauthenticated},
		{errkit.CodePermissionDenied, codes.PermissionDenied},
		{errkit.CodeUnavailable, codes.Unavailable},
		{errkit.CodeDeadlineExceeded, codes.DeadlineExceeded},
		{errkit.CodeCanceled, codes.Canceled},
		{errkit.CodeInternal, codes.Internal},
		{errkit.CodeUnknown, codes.Unknown},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.code), func(t *testing.T) {
			t.Parallel()
			got := grpcerr.ToGRPCStatus(errkit.New(errkit.WithCode(tc.code), errkit.WithMessage("hi")))
			if got.Code() != tc.want {
				t.Errorf("default: want %v, got %v", tc.want, got.Code())
			}
			if got.Message() != "hi" {
				t.Errorf("default message: want %q, got %q", "hi", got.Message())
			}
		})
	}
}

func TestToGRPCStatus_Fallback(t *testing.T) {
	t.Parallel()
	// nil and non-errkit and unmapped all go to Unknown — the safest
	// default for a gRPC handler.
	if got := grpcerr.ToGRPCStatus(nil); got.Code() != codes.Unknown {
		t.Errorf("nil err -> Unknown, got %v", got.Code())
	}
	if got := grpcerr.ToGRPCStatus(errors.New("plain")); got.Code() != codes.Unknown {
		t.Errorf("non-errkit -> Unknown, got %v", got.Code())
	}
	if got := grpcerr.ToGRPCStatus(errkit.New(errkit.WithCode(errkit.Code("PAYMENT_REQUIRED")))); got.Code() != codes.Unknown {
		t.Errorf("unmapped -> Unknown, got %v", got.Code())
	}
}

func TestMapper_Override(t *testing.T) {
	t.Parallel()
	m := grpcerr.NewMapper(map[errkit.Code]codes.Code{
		errkit.CodeInvalidArgument: codes.FailedPrecondition,
	})
	got := m.ToGRPCStatus(errkit.New(errkit.WithCode(errkit.CodeInvalidArgument)))
	if got.Code() != codes.FailedPrecondition {
		t.Errorf("override not applied: got %v", got.Code())
	}
}

func TestToGRPCError_RoundTrip(t *testing.T) {
	t.Parallel()
	in := errkit.NotFound("nope")
	out := grpcerr.ToGRPCError(in)
	st, ok := status.FromError(out)
	if !ok {
		t.Fatalf("ToGRPCError result is not a status: %v", out)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("decoded code: want NotFound, got %v", st.Code())
	}
	if st.Message() != "nope" {
		t.Errorf("decoded message: want %q, got %q", "nope", st.Message())
	}
}
