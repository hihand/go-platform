package responsekit

import (
	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/errkit/httperr"
)

// successEnvelope wraps data in the canonical success envelope:
// {"data": data}. Used by every adapter.
func successEnvelope(data any) Envelope {
	return Envelope{Data: data}
}

// errorEnvelope builds the canonical error envelope from any
// error. Behaviour:
//
//   - nil → {"code":"INTERNAL","message":""}
//   - errkit.Error (anywhere in the cause chain) → its Code + Message
//   - anything else → {"code":"INTERNAL","message":err.Error()}
//
// The internal-code fallback for the last two cases is intentional:
// a raw Go error reaching the HTTP boundary is treated as a server
// bug, so the public code is the generic INTERNAL while the
// underlying err.Error() is preserved for the client to log.
func errorEnvelope(err error) ErrorEnvelope {
	if err == nil {
		return ErrorEnvelope{Error: ErrorBody{Code: string(errkit.CodeInternal), Message: ""}}
	}
	if e, ok := errkit.FromError(err); ok {
		return ErrorEnvelope{Error: ErrorBody{Code: string(e.Code()), Message: e.Message()}}
	}
	return ErrorEnvelope{Error: ErrorBody{Code: string(errkit.CodeInternal), Message: err.Error()}}
}

// statusCode maps any error to an HTTP status code via
// errkit/httperr. nil and non-errkit errors fall back to 500.
func statusCode(err error) int {
	return httperr.StatusCode(err)
}