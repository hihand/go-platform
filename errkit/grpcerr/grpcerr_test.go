package grpcerr_test

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/errkit/grpcerr"
)

// defaultMapping exhaustively pins every Code the library maps by default.
// If you add a Code or change a mapping, add the row here so the wire
// contract stays explicit.
func TestToGRPCStatus_DefaultMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code errkit.Code
		want codes.Code
	}{
		// Transport / lifecycle
		{errkit.CodeCanceled, codes.Canceled},
		{errkit.CodeDeadlineExceeded, codes.DeadlineExceeded},
		{errkit.CodeRequestTimeout, codes.DeadlineExceeded},

		// Client errors → InvalidArgument family
		{errkit.CodeInvalidArgument, codes.InvalidArgument},
		{errkit.CodeUnprocessableEntity, codes.InvalidArgument},
		{errkit.CodeMethodNotAllowed, codes.InvalidArgument},
		{errkit.CodeURITooLong, codes.InvalidArgument},
		{errkit.CodeExpectationFailed, codes.InvalidArgument},
		{errkit.CodeMisdirectedRequest, codes.InvalidArgument},
		{errkit.CodeNotAcceptable, codes.InvalidArgument},
		{errkit.CodeLengthRequired, codes.InvalidArgument},
		{errkit.CodeUnsupportedMediaType, codes.InvalidArgument},

		// Client errors → Unauthenticated / PermissionDenied
		{errkit.CodeUnauthenticated, codes.Unauthenticated},
		{errkit.CodePermissionDenied, codes.PermissionDenied},

		// Client errors → FailedPrecondition
		{errkit.CodeLocked, codes.FailedPrecondition},
		{errkit.CodeFailedDependency, codes.FailedPrecondition},
		{errkit.CodeUnavailableForLegalReasons, codes.FailedPrecondition},
		{errkit.CodePreconditionFailed, codes.FailedPrecondition},

		// Client errors → NotFound / AlreadyExists / Aborted
		{errkit.CodeNotFound, codes.NotFound},
		{errkit.CodeGone, codes.NotFound},
		{errkit.CodeAlreadyExists, codes.AlreadyExists},
		{errkit.CodeConflict, codes.Aborted},

		// Client errors → OutOfRange
		{errkit.CodeRangeNotSatisfiable, codes.OutOfRange},

		// Client errors → ResourceExhausted
		{errkit.CodeTooManyRequests, codes.ResourceExhausted},
		{errkit.CodePayloadTooLarge, codes.ResourceExhausted},
		{errkit.CodeRequestHeaderFieldsTooLarge, codes.ResourceExhausted},

		// Server errors
		{errkit.CodeInternal, codes.Internal},
		{errkit.CodeNotImplemented, codes.Unimplemented},
		{errkit.CodeBadGateway, codes.Unavailable},
		{errkit.CodeUnavailable, codes.Unavailable},
		{errkit.CodeDataLoss, codes.DataLoss},
		{errkit.CodeNetworkAuthenticationRequired, codes.Unauthenticated},

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
	for _, code := range []errkit.Code{
		errkit.CodeDuplicate,        // built-in but intentionally unmapped
		errkit.CodePaymentRequired,  // built-in but intentionally unmapped
		errkit.CodeUpgradeRequired,  // built-in but intentionally unmapped
		errkit.Code("PAYMENT_REQUIRED"),
	} {
		if got := grpcerr.ToGRPCStatus(errkit.New(errkit.WithCode(code))); got.Code() != codes.Unknown {
			t.Errorf("unmapped code %q -> Unknown, got %v", code, got.Code())
		}
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

// Mapper is free to introduce mappings for codes the default table skips
// (CodeDuplicate, CodePaymentRequired, …). Lock that path in.
func TestMapper_AddsNewEntries(t *testing.T) {
	t.Parallel()
	m := grpcerr.NewMapper(map[errkit.Code]codes.Code{
		errkit.CodeDuplicate: codes.AlreadyExists,
	})
	if got := m.ToGRPCStatus(errkit.New(errkit.WithCode(errkit.CodeDuplicate))); got.Code() != codes.AlreadyExists {
		t.Errorf("CodeDuplicate override not applied: got %v", got.Code())
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
