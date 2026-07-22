package errkit

import "errors"

// Code is a stable, machine-friendly identifier for an error. Custom codes
// are allowed; the constants below cover the most common cases.
//
// Codes are compared by value, so callers may freely declare their own.
//
//	const errkit.CodePaymentRequired errkit.Code = "PAYMENT_REQUIRED"
type Code string

// Well-known codes. Each block is grouped by lifecycle stage so the same
// status family (e.g. 4xx vs 5xx) lives next to its siblings.
//
// Naming rationale:
//
//   - The wire label follows gRPC conventions ("INVALID_ARGUMENT",
//     "INTERNAL") rather than HTTP-internal names ("InternalServerError",
//     "BadRequest"), so the same string round-trips cleanly through both
//     the httperr and grpcerr adapters.
//   - Where several codes share the same HTTP/gRPC slot — e.g.
//     "ALREADY_EXISTS", "CONFLICT", "DUPLICATE" all map to HTTP 409 —
//     the distinction lives on the errkit side (semantic clarity for
//     logs / metadata) while the adapter picks the right wire status.
//   - Custom codes are first-class: declare them as needed (e.g.
//     `const CodePaymentRequired Code = "PAYMENT_REQUIRED"`).
//
// Codes that the built-in adapters do not recognise — including every
// custom code, and a few intentionally-unmapped built-ins such as
// `CodeDuplicate` — fall back to the protocol's generic internal error.
const (
	// ----- Transport / lifecycle -------------------------------------

	// CodeUnknown is the zero-value fallback. Do not raise it directly;
	// it is what callers see when a non-errkit error reaches the adapter
	// or when a caller forgot `WithCode(...)`.
	CodeUnknown Code = "UNKNOWN"

	// CodeCanceled signals the request was aborted by the caller
	// (HTTP 499, gRPC Canceled).
	CodeCanceled Code = "CANCELED"

	// CodeDeadlineExceeded signals the operation did not finish before
	// its deadline (HTTP 504, gRPC DeadlineExceeded).
	CodeDeadlineExceeded Code = "DEADLINE_EXCEEDED"

	// CodeRequestTimeout is the HTTP-specific flavour of
	// "request ran out of time on the client/server edge".
	// It surfaces as 408; gRPC has no dedicated code and falls back
	// to DeadlineExceeded on the wire.
	CodeRequestTimeout Code = "REQUEST_TIMEOUT"

	// ----- Client errors (4xx) ---------------------------------------

	// CodeInvalidArgument — generic 400. "The request was malformed."
	CodeInvalidArgument Code = "INVALID_ARGUMENT"

	// CodeUnauthenticated — 401. "Credentials are missing or invalid."
	CodeUnauthenticated Code = "UNAUTHENTICATED"

	// CodePermissionDenied — 403. "Credentials are valid but the caller
	// is not allowed to touch this resource."
	CodePermissionDenied Code = "PERMISSION_DENIED"

	// CodeNotFound — 404.
	CodeNotFound Code = "NOT_FOUND"

	// CodeMethodNotAllowed — 405. The resource exists but doesn't support
	// the verb used.
	CodeMethodNotAllowed Code = "METHOD_NOT_ALLOWED"

	// CodeNotAcceptable — 406. The server cannot produce a response
	// matching the Accept headers.
	CodeNotAcceptable Code = "NOT_ACCEPTABLE"

	// CodeConflict — generic 409. "The request clashes with the current
	// state of the resource." Use it for business-rule conflicts that
	// aren't strictly "another writer already created it".
	CodeConflict Code = "CONFLICT"

	// CodeAlreadyExists — gRPC-flavoured 409. "The resource was created
	// by another writer while this request was in flight."
	CodeAlreadyExists Code = "ALREADY_EXISTS"

	// CodeDuplicate — 409. Strictly stronger than AlreadyExists:
	// a uniqueness constraint (DB unique index, idempotency key) was
	// violated. The HTTP/gRPC adapters intentionally do not map this
	// — pick a status via `NewMapper`.
	CodeDuplicate Code = "DUPLICATE"

	// CodeGone — 410. The resource was here once but is permanently gone.
	CodeGone Code = "GONE"

	// CodePayloadTooLarge — 413 / gRPC ResourceExhausted on the wire.
	// The body is over the documented upper bound.
	CodePayloadTooLarge Code = "PAYLOAD_TOO_LARGE"

	// CodeURITooLong — 414. The request URI exceeds the server's limit.
	CodeURITooLong Code = "URI_TOO_LONG"

	// CodeUnsupportedMediaType — 415. The body Content-Type is not
	// accepted by this endpoint.
	CodeUnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE"

	// CodeRangeNotSatisfiable — 416. The Range header was malformed or
	// exceeded the resource length.
	CodeRangeNotSatisfiable Code = "RANGE_NOT_SATISFIABLE"

	// CodeExpectationFailed — 417. Sent when the Expect header
	// (typically 100-continue) cannot be honoured.
	CodeExpectationFailed Code = "EXPECTATION_FAILED"

	// CodeMisdirectedRequest — 421. The server cannot produce a
	// response for the URI on this connection.
	CodeMisdirectedRequest Code = "MISDIRECTED_REQUEST"

	// CodeUnprocessableEntity — 422. The request is syntactically valid
	// (so InvalidArgument is wrong) but semantically rejected — e.g.
	// a business-rule failure caught at validation time.
	CodeUnprocessableEntity Code = "UNPROCESSABLE_ENTITY"

	// CodeLocked — 423. WebDAV-flavoured 409 ("Conflict (Locked)").
	// The resource is currently locked.
	CodeLocked Code = "LOCKED"

	// CodeFailedDependency — 424. WebDAV: the request failed because
	// a previous part of the same compound request failed.
	CodeFailedDependency Code = "FAILED_DEPENDENCY"

	// CodeTooManyRequests — 429. Generic rate-limit bucket.
	CodeTooManyRequests Code = "TOO_MANY_REQUESTS"

	// CodeRequestHeaderFieldsTooLarge — 431. Header section exceeded
	// the server's limit.
	CodeRequestHeaderFieldsTooLarge Code = "REQUEST_HEADER_FIELDS_TOO_LARGE"

	// CodeUnavailableForLegalReasons — 451. Censorship / takedown.
	CodeUnavailableForLegalReasons Code = "UNAVAILABLE_FOR_LEGAL_REASONS"

	// CodePaymentRequired — 402. Reserved by RFC 9110. Use it when the
	// caller must pay to unlock the request (subscriptions, metered
	// APIs). The adapters do not map it by default because 402 is
	// historically ambiguous — wire it explicitly via `NewMapper`.
	CodePaymentRequired Code = "PAYMENT_REQUIRED"

	// CodePreconditionFailed — 412. An If-Match / If-None-Match /
	// If-Modified-Since check failed.
	CodePreconditionFailed Code = "PRECONDITION_FAILED"

	// CodeLengthRequired — 411. The request must carry a Content-Length.
	CodeLengthRequired Code = "LENGTH_REQUIRED"

	// CodeUpgradeRequired — 426. The client should switch protocols
	// (e.g. HTTP → WebSocket).
	CodeUpgradeRequired Code = "UPGRADE_REQUIRED"

	// ----- Server errors (5xx) ---------------------------------------

	// CodeInternal — the generic "something went wrong on our side".
	// Surfaces as HTTP 500 / gRPC `Internal`. The previous label
	// "INTERNAL_SERVER_ERROR" was dropped to align with gRPC and to
	// avoid the misleading "this is always a server bug" reading.
	CodeInternal Code = "INTERNAL"

	// CodeNotImplemented — 501.
	CodeNotImplemented Code = "NOT_IMPLEMENTED"

	// CodeBadGateway — 502 / gRPC Unavailable.
	CodeBadGateway Code = "BAD_GATEWAY"

	// CodeUnavailable — 503 / gRPC Unavailable. "Temporary outage,
	// try again later."
	CodeUnavailable Code = "UNAVAILABLE"

	// CodeDataLoss — 507 / gRPC DataLoss. "Recoverable data was
	// destroyed or corrupted; the request cannot be retried."
	CodeDataLoss Code = "DATA_LOSS"

	// CodeNetworkAuthenticationRequired — 511. Captive portal / proxy.
	CodeNetworkAuthenticationRequired Code = "NETWORK_AUTHENTICATION_REQUIRED"
)

// Code returns the Code for this error. For an empty *impl (which New never
// returns) the zero value is Code("") which callers should treat the same as
// CodeUnknown.
func (i *impl) Code() Code {
	return i.code
}

// CodeOf extracts the Code from an arbitrary error, walking the cause chain
// via errors.As. If no errkit error is present in the chain CodeUnknown is
// returned.
func CodeOf(err error) Code {
	if err == nil {
		return CodeUnknown
	}
	var e Error
	if errors.As(err, &e) {
		return e.Code()
	}
	return CodeUnknown
}

// FromError returns the first errkit.Error encountered while walking the
// cause chain, plus a boolean indicating whether one was found.
func FromError(err error) (Error, bool) {
	if err == nil {
		return nil, false
	}
	var e Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// IsCode reports whether err (or any wrapped cause) is an errkit error whose
// Code equals code. The comparison is value-based and is safe to use with
// custom Code constants.
func IsCode(err error, code Code) bool {
	c, ok := FromError(err)
	if !ok {
		return false
	}
	return c.Code() == code
}
