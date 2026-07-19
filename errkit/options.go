package errkit

import "maps"

// Option mutates the internal config used during error construction. Options
// are applied in the order they are supplied; later options override earlier
// ones for scalar fields. Map-style options (WithMetadata) shallow-copy their
// input so callers may safely reuse the map.
type Option func(*impl)

// WithCode sets the machine-friendly Code on the constructed error. If no
// WithCode is supplied to New, the code defaults to CodeUnknown.
func WithCode(code Code) Option {
	return func(i *impl) {
		i.code = code
	}
}

// WithMessage sets the human-readable Message on the constructed error. It
// is not a special parameter; everything in errkit flows through Options.
func WithMessage(msg string) Option {
	return func(i *impl) {
		i.message = msg
	}
}

// WithCause attaches err as the underlying cause. It is returned from
// Unwrap() and used by errors.Is / errors.As.
//
// Wrapping a non-errkit error with New(WithCause(err)) is supported: the
// resulting error keeps the errkit Code/Message on top while remaining
// errors.Is-compatible with err.
func WithCause(err error) Option {
	return func(i *impl) {
		i.cause = err
	}
}

// WithMetadata attaches a JSON-friendly metadata map. The supplied map is
// shallow-copied at construction time (via the stdlib maps.Clone) and a
// fresh copy is returned by Metadata() on every call, so callers can mutate
// their working map freely without affecting the constructed error.
func WithMetadata(m map[string]any) Option {
	return func(i *impl) {
		if len(m) == 0 {
			return
		}
		i.metadata = maps.Clone(m)
	}
}

// WithStack opts the constructed error into stack-frame recording. The
// captured frames are stored on an unexported field of *impl and are
// reserved for a future hardened public API.
//
// Today the call is essentially free if you forget it: sugar constructors do
// not invoke WithStack, so hot paths stay zero-allocation. When the
// public-facing StackTrace() accessor lands it will be additive and will
// not break source compatibility.
func WithStack() Option {
	return func(err *impl) {
		err.stack = captureStack(2)
	}
}