package contextkit

import "context"

// Request carries request-scoped identifiers that are useful to
// propagate from the transport edge (HTTP/gRPC handler, queue worker,
// CLI runner) into the call graph below it.
//
// All fields are plain strings so the struct stays cheap to copy, has
// no allocation surprises, and never holds pointers that could be
// shared between goroutines. Tracing and span identifiers live here as
// plain strings only — the package does not introduce any tracing
// types; callers fill these in from whatever tracing system they use.
type Request struct {
	// RequestID is a transport- or platform-issued identifier for
	// the inbound request. Optional.
	RequestID string

	// TraceID is the identifier of the enclosing trace, if any.
	// Optional.
	TraceID string

	// SpanID is the identifier of the current span within the
	// trace, if any. Optional.
	SpanID string
}

// WithRequest returns a derived context that carries req. Calling it
// more than once replaces the previous Request on the returned
// context; the original context is not modified.
//
// A nil context is treated as context.Background() so the helper is
// safe to call from places that have not yet prepared a context
// (e.g. helpers invoked from tests).
func WithRequest(ctx context.Context, req Request) context.Context {
	return ctxWithValue(ctx, requestKey{}, req)
}

// GetRequest returns the Request stored on ctx, or a zero Request if
// none has been stored. The returned value is a copy; mutating it has
// no effect on the context.
//
// A nil context is treated like an empty context: the call returns
// the zero Request and never panics.
func GetRequest(ctx context.Context) Request {
	r, _ := ctxValue[Request](ctx, requestKey{})
	return r
}
