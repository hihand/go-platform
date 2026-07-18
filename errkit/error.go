package errkit

import "fmt"

// Error implements the error interface. The format is stable and machine
// parseable:
//
//	CODE: message
//	CODE: message: cause.Error()
//
// Adapters and tests rely on this exact prefix for the "CODE" portion.
func (i *impl) Error() string {
	if i.cause != nil {
		return fmt.Sprintf("%s: %s: %s", i.code, i.message, i.cause.Error())
	}
	return fmt.Sprintf("%s: %s", i.code, i.message)
}

// Unwrap returns the underlying cause. This is the single hook used by
// errors.Is and errors.As to walk the cause chain.
//
// errkit intentionally does not expose a separate Cause() method: callers
// who want the immediate cause should call errors.Unwrap(err) (Go 1.20+)
// or call Unwrap directly via the Error interface.
func (i *impl) Unwrap() error {
	return i.cause
}
