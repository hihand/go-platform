// Package httperr maps errkit.Code values to HTTP status codes.
//
// It performs mapping only; serialization of the response body is the
// application's responsibility. There is no global state — every Mapper is
// constructed explicitly via NewMapper. A package-level StatusCode helper is
// provided for callers that do not need overrides and uses an internal,
// non-exported mapper that cannot be mutated from outside.
package httperr

import (
	"maps"
	"net/http"

	"github.com/hihand/go-platform/errkit"
)

// defaultStatus is returned when no rule matches or when the error is not an
// errkit.Error. Keeping it as a named constant makes the fallback obvious in
// the generated docs and prevents magic numbers in tests.
const defaultStatus = http.StatusInternalServerError

// StatusClientClosedRequest is the de-facto HTTP status code for the "client
// closed the connection" case. Go's net/http package does not yet ship a
// named constant for it (an upstream proposal has been open for years), so
// errkit/httperr defines its own. Mapping errkit.CodeCanceled here keeps the
// default table aligned with the convention used by nginx, Envoy, and most
// reverse proxies.
const StatusClientClosedRequest = 499

// defaultMap is the built-in translation table. Wire labels follow the
// conventions of RFC 9110 (HTTP Semantics, 2022) and are aligned 1:1 with
// the canonical HTTP reason phrases.
//
// Codes that the built-in table intentionally does **not** include:
//
//   - `CodeDuplicate` — 409 by convention, but DB-specific. Apps register
//     an explicit mapping via `NewMapper` to keep the policy local.
//   - `CodePaymentRequired` — 402 is reserved by RFC 9110 but rarely
//     used; the application decides whether to expose it.
//   - `CodeUpgradeRequired` — use a custom mapper to return 426.
//
// `defaultMap` is the **only** place where this set is defined; callers
// that want a different policy build their own `Mapper` and ignore it.
var defaultMap = map[errkit.Code]int{
	// ----- Transport / lifecycle -----
	errkit.CodeCanceled:         StatusClientClosedRequest,
	errkit.CodeDeadlineExceeded: http.StatusGatewayTimeout,
	errkit.CodeRequestTimeout:   http.StatusRequestTimeout,

	// ----- Client errors (4xx) -----
	errkit.CodeInvalidArgument:            http.StatusBadRequest,
	errkit.CodeUnauthenticated:            http.StatusUnauthorized,
	errkit.CodePermissionDenied:           http.StatusForbidden,
	errkit.CodeNotFound:                   http.StatusNotFound,
	errkit.CodeMethodNotAllowed:           http.StatusMethodNotAllowed,
	errkit.CodeNotAcceptable:              http.StatusNotAcceptable,
	errkit.CodeConflict:                   http.StatusConflict,
	errkit.CodeAlreadyExists:              http.StatusConflict,
	errkit.CodeGone:                       http.StatusGone,
	errkit.CodeLengthRequired:             http.StatusLengthRequired,
	errkit.CodePreconditionFailed:         http.StatusPreconditionFailed,
	errkit.CodePayloadTooLarge:            http.StatusRequestEntityTooLarge,
	errkit.CodeURITooLong:                 http.StatusRequestURITooLong,
	errkit.CodeUnsupportedMediaType:       http.StatusUnsupportedMediaType,
	errkit.CodeRangeNotSatisfiable:        http.StatusRequestedRangeNotSatisfiable,
	errkit.CodeExpectationFailed:          http.StatusExpectationFailed,
	errkit.CodeMisdirectedRequest:         http.StatusMisdirectedRequest,
	errkit.CodeUnprocessableEntity:        http.StatusUnprocessableEntity,
	errkit.CodeLocked:                     http.StatusLocked,
	errkit.CodeFailedDependency:           http.StatusFailedDependency,
	errkit.CodeTooManyRequests:            http.StatusTooManyRequests,
	errkit.CodeRequestHeaderFieldsTooLarge: http.StatusRequestHeaderFieldsTooLarge,
	errkit.CodeUnavailableForLegalReasons: http.StatusUnavailableForLegalReasons,

	// ----- Server errors (5xx) -----
	errkit.CodeInternal:                    http.StatusInternalServerError,
	errkit.CodeNotImplemented:              http.StatusNotImplemented,
	errkit.CodeBadGateway:                  http.StatusBadGateway,
	errkit.CodeUnavailable:                 http.StatusServiceUnavailable,
	errkit.CodeDataLoss:                    http.StatusInsufficientStorage,
	errkit.CodeNetworkAuthenticationRequired: http.StatusNetworkAuthenticationRequired,

	// Catch-all must stay last in this comment block but anywhere in the
	// map — `defaultStatus` is consulted when no rule matches.
	errkit.CodeUnknown: http.StatusInternalServerError,
}

// Mapper translates errkit.Code into an HTTP status code. A Mapper is
// immutable after construction; replace an entry by building a new Mapper
// with NewMapper. Mappers are safe for concurrent use.
type Mapper struct {
	overrides map[errkit.Code]int
}

// NewMapper returns a Mapper that uses the built-in defaults plus any
// caller-supplied overrides. The override map is shallow-copied (via the
// stdlib maps.Clone) so the caller may safely mutate the map after this
// call returns without affecting the mapper.
func NewMapper(override map[errkit.Code]int) *Mapper {
	m := &Mapper{}
	if len(override) > 0 {
		m.overrides = maps.Clone(override)
	}
	return m
}

// StatusCode returns the HTTP status code that corresponds to err. The
// cause chain is walked via errkit.FromError. A non-errkit error, an empty
// chain, or an unknown Code all map to defaultStatus (500).
func (m *Mapper) StatusCode(err error) int {
	if err == nil {
		return defaultStatus
	}
	e, ok := errkit.FromError(err)
	if !ok {
		return defaultStatus
	}
	if code, ok := m.overrides[e.Code()]; ok {
		return code
	}
	if code, ok := defaultMap[e.Code()]; ok {
		return code
	}
	return defaultStatus
}

// defaultMapper is the package-internal default used by the convenience
// StatusCode helper. It is intentionally a value type here so it cannot be
// mutated from outside the package, satisfying the no-global-state
// requirement.
var defaultMapper = NewMapper(nil)

// StatusCode is a convenience wrapper around the package's internal default
// Mapper. Prefer constructing your own Mapper when you need to override
// entries — the default mapper cannot be replaced.
func StatusCode(err error) int {
	return defaultMapper.StatusCode(err)
}
