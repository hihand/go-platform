package logkit

import "context"

// ContextMapper is the single seam between logkit and the application's
// context. The logger knows nothing about request IDs, trace IDs, user
// IDs, tenant IDs, or any other transport- or framework-specific
// correlation field. Instead, when *Context methods are called the
// logger invokes the configured mapper, passing it the context the
// caller supplied and a scratch slice, and merges the returned slice
// into the record.
//
// The mapper is append-style:
//
//	func(ctx context.Context, attrs []Attr) []Attr
//
// It returns the (possibly grown) slice. Returning the input unchanged
// is the no-op signal. The append shape lets the caller reuse the
// underlying capacity across mappers and avoids the alloc that a
// "return []Attr{...}" signature would force on every call.
//
// The mapper type is intentionally a function rather than an
// interface: a function is the smallest possible abstraction and
// carries no inherent state. Composition (chaining multiple mappers)
// is trivially expressed with a small wrapper at the call site.
//
// If no mapper is configured, the logger ignores the context entirely.
// Logger construction is unaffected: zero allocations, zero behavioural
// cost, zero global state.
//
// Typical wiring:
//
//	logger := logkit.New(
//	    logkit.WithContextMapper(myApp.ExtractLogAttrs),
//	)
//
// where myApp.ExtractLogAttrs is owned by the application (or a
// separate package such as otelkit) and knows how to read whatever
// keys it has previously stored in the context.
type ContextMapper func(ctx context.Context, attrs []Attr) []Attr

// WithContextMapper installs a ContextMapper on the Logger. It is
// called once at construction time; the field is read-only thereafter
// so concurrent log calls are race-free.
//
// A nil mapper is a no-op so callers can safely forward optional
// configuration without a separate guard.
func WithContextMapper(mapper ContextMapper) Option {
	return func(l *impl) {
		if mapper == nil {
			return
		}
		l.mapper = mapper
	}
}