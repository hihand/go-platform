// Package grpcerr maps errkit.Code values to gRPC status codes and builds
// google.golang.org/grpc/status.Status values from errkit errors.
//
// It performs mapping only; wrapping the result into a gRPC trailer or
// sending it on the wire is the application's responsibility. There is no
// global state — every Mapper is constructed explicitly via NewMapper. A
// package-level ToGRPCStatus helper is provided for callers that do not
// need overrides.
package grpcerr

import (
	"maps"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hihand/go-platform/errkit"
)

// defaultCode is returned when no rule matches or the error is not an
// errkit.Error. Per gRPC convention Unknown is the catch-all.
const defaultCode = codes.Unknown

// defaultMap is the built-in translation table. Wire labels follow the gRPC
// status codes document (https://grpc.io/docs/grpc-framework/status-codes/).
//
// Several errkit codes collapse onto the same gRPC code because the
// protocol cannot express the distinction. The notable cases:
//
//   - `CodeInvalidArgument`, `CodeUnprocessableEntity`, `CodeURITooLong`,
//     `CodeMethodNotAllowed`, `CodeExpectationFailed`, `CodeMisdirectedRequest`
//     → `codes.InvalidArgument`: validation/structural failures.
//   - `CodePermissionDenied`, `CodeLocked`, `CodeFailedDependency`,
//     `CodeUnavailableForLegalReasons` → `codes.FailedPrecondition`:
//     the operation could succeed on a different state.
//   - `CodeTooManyRequests`, `CodePayloadTooLarge`,
//     `CodeRequestHeaderFieldsTooLarge` → `codes.ResourceExhausted`.
//   - `CodeUnavailable`, `CodeBadGateway`, `CodeNetworkAuthenticationRequired`
//     → `codes.Unavailable`.
//   - `CodeRangeNotSatisfiable`, `CodeLengthRequired` → `codes.OutOfRange`.
//
// Codes the table intentionally omits: `CodeDuplicate`, `CodeUpgradeRequired`,
// `CodePaymentRequired`, `CodeRequestTimeout` (gRPC has no `RequestTimeout`;
// we map to `DeadlineExceeded` for consistency). Callers that need an exact
// mapping register it via `NewMapper`.
var defaultMap = map[errkit.Code]codes.Code{
	// ----- Transport / lifecycle -----
	errkit.CodeCanceled:         codes.Canceled,
	errkit.CodeDeadlineExceeded: codes.DeadlineExceeded,
	errkit.CodeRequestTimeout:   codes.DeadlineExceeded,

	// ----- Client errors (4xx → request-shape failures) -----
	errkit.CodeInvalidArgument:            codes.InvalidArgument,
	errkit.CodeUnprocessableEntity:        codes.InvalidArgument,
	errkit.CodeMethodNotAllowed:           codes.InvalidArgument,
	errkit.CodeURITooLong:                 codes.InvalidArgument,
	errkit.CodeExpectationFailed:          codes.InvalidArgument,
	errkit.CodeMisdirectedRequest:         codes.InvalidArgument,

	// ----- Client errors → FailedPrecondition -----
	errkit.CodePermissionDenied:             codes.PermissionDenied,
	errkit.CodeUnauthenticated:             codes.Unauthenticated,
	errkit.CodeLocked:                      codes.FailedPrecondition,
	errkit.CodeFailedDependency:            codes.FailedPrecondition,
	errkit.CodeUnavailableForLegalReasons:  codes.FailedPrecondition,

	errkit.CodeNotFound:        codes.NotFound,
	errkit.CodeAlreadyExists:   codes.AlreadyExists,
	errkit.CodeConflict:        codes.Aborted,
	errkit.CodeGone:            codes.NotFound,
	errkit.CodeNotAcceptable:   codes.InvalidArgument,
	errkit.CodeLengthRequired:  codes.InvalidArgument,
	errkit.CodePreconditionFailed: codes.FailedPrecondition,
	errkit.CodeUnsupportedMediaType: codes.InvalidArgument,
	errkit.CodeRangeNotSatisfiable: codes.OutOfRange,

	// ----- Client errors → resource exhaustion -----
	errkit.CodeTooManyRequests:             codes.ResourceExhausted,
	errkit.CodePayloadTooLarge:             codes.ResourceExhausted,
	errkit.CodeRequestHeaderFieldsTooLarge: codes.ResourceExhausted,

	// ----- Server errors (5xx) -----
	errkit.CodeInternal:                    codes.Internal,
	errkit.CodeNotImplemented:              codes.Unimplemented,
	errkit.CodeBadGateway:                  codes.Unavailable,
	errkit.CodeUnavailable:                 codes.Unavailable,
	errkit.CodeDataLoss:                    codes.DataLoss,
	errkit.CodeNetworkAuthenticationRequired: codes.Unauthenticated,

	// Catch-all.
	errkit.CodeUnknown: codes.Unknown,
}

// Mapper translates errkit.Code into a gRPC code. A Mapper is immutable
// after construction; replace an entry by building a new Mapper with
// NewMapper. Mappers are safe for concurrent use.
type Mapper struct {
	overrides map[errkit.Code]codes.Code
}

// NewMapper returns a Mapper that uses the built-in defaults plus any
// caller-supplied overrides. The override map is shallow-copied (via the
// stdlib maps.Clone) so the caller may safely mutate the map after this
// call returns.
func NewMapper(override map[errkit.Code]codes.Code) *Mapper {
	m := &Mapper{}
	if len(override) > 0 {
		m.overrides = maps.Clone(override)
	}
	return m
}

// grpcCode returns the gRPC code for err; the same fallback rules as
// Mapper.StatusCode apply.
func (m *Mapper) grpcCode(err error) codes.Code {
	if err == nil {
		return defaultCode
	}
	e, ok := errkit.FromError(err)
	if !ok {
		return defaultCode
	}
	if c, ok := m.overrides[e.Code()]; ok {
		return c
	}
	if c, ok := defaultMap[e.Code()]; ok {
		return c
	}
	return defaultCode
}

// ToGRPCStatus builds a *status.Status from err. The Status.Code comes from
// the Mapper's table (falling back to codes.Unknown). The Status.Message
// comes from errkit.MessageOf — i.e. the errkit message, not the underlying
// cause's text. When err is nil, a status with codes.Unknown and empty message
// is returned so the helper can be used unconditionally.
func (m *Mapper) ToGRPCStatus(err error) *status.Status {
	c := m.grpcCode(err)
	return status.New(c, errkit.MessageOf(err))
}

var defaultMapper = NewMapper(nil)

// ToGRPCStatus is a convenience wrapper around the package's internal
// default Mapper. Prefer constructing your own Mapper when you need to
// override entries — the default mapper cannot be replaced.
func ToGRPCStatus(err error) *status.Status {
	return defaultMapper.ToGRPCStatus(err)
}

// ToGRPCError returns the error form of status.Status (i.e. the result of
// (*status.Status).Err()) so it can be returned directly from a gRPC
// handler implementation.
//
//	err := grpcerr.ToGRPCError(errkit.NotFound("user 42"))
//	return err
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	return ToGRPCStatus(err).Err()
}
