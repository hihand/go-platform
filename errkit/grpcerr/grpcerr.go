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

// defaultMap is the built-in translation table. Mirrors common practice:
// validation -> InvalidArgument, not-found -> NotFound, etc.
var defaultMap = map[errkit.Code]codes.Code{
	errkit.CodeInvalidArgument:  codes.InvalidArgument,
	errkit.CodeNotFound:         codes.NotFound,
	errkit.CodeAlreadyExists:    codes.AlreadyExists,
	errkit.CodeUnauthenticated:  codes.Unauthenticated,
	errkit.CodePermissionDenied: codes.PermissionDenied,
	errkit.CodeUnavailable:      codes.Unavailable,
	errkit.CodeDeadlineExceeded: codes.DeadlineExceeded,
	errkit.CodeCanceled:         codes.Canceled,
	errkit.CodeInternal:         codes.Internal,
	errkit.CodeUnknown:          codes.Unknown,
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
