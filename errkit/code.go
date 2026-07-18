package errkit

import "errors"

// Code is a stable, machine-friendly identifier for an error. Custom codes
// are allowed; the constants below cover the most common cases.
//
// Codes are compared by value, so callers may freely declare their own.
//
//	type errkit.Code = errkit.Code  // aliasing is not necessary
//	const errkit.CodePaymentRequired errkit.Code = "PAYMENT_REQUIRED"
type Code string

// Well-known codes. Other adapters (httperr, grpcerr, graphqlerr) ship a
// default mapping that recognises this set; unknown codes fall back to
// the protocol's "internal" status.
const (
	CodeUnknown          Code = "UNKNOWN"
	CodeInvalidArgument  Code = "INVALID_ARGUMENT"
	CodeNotFound         Code = "NOT_FOUND"
	CodeAlreadyExists    Code = "ALREADY_EXISTS"
	CodeConflict         Code = "CONFLICT"
	CodeDuplicate        Code = "DUPLICATE"
	CodeUnauthenticated  Code = "UNAUTHENTICATED"
	CodePermissionDenied Code = "PERMISSION_DENIED"
	CodeInternal         Code = "INTERNAL"
	CodeUnavailable      Code = "UNAVAILABLE"
	CodeDeadlineExceeded Code = "DEADLINE_EXCEEDED"
	CodeCanceled         Code = "CANCELED"
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
