package contextkit

import "context"

// requestKey is the unexported context key under which the package
// stores a Request value. It is a zero-sized struct so the key is a
// unique, comparable identity without paying any allocation cost.
//
// The type is deliberately unexported: any collision with keys defined
// outside this package would be a programmer error, and exporting it
// would invite that collision.
type requestKey struct{}

// identityKey is the unexported context key under which the package
// stores the generic identity value. Like requestKey it is a
// zero-sized struct used for identity comparison only.
type identityKey struct{}

// ctxWithValue is a thin, exported-by-import wrapper around
// context.WithValue that hides the key type from the call site. It is
// used by both request.go and identity.go so the stdlib allocation and
// not-found semantics live in exactly one place.
//
// When ctx is nil we substitute context.Background() so the call to
// context.WithValue never panics on a nil receiver.
func ctxWithValue(ctx context.Context, key any, value any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, key, value)
}

// ctxValue is the symmetric read-side helper. It handles the nil-
// context and missing-value cases uniformly so every getter in the
// package follows the same "tolerate nil, return zero on miss" rule.
func ctxValue[T any](ctx context.Context, key any) (T, bool) {
	var zero T
	if ctx == nil {
		return zero, false
	}
	v, ok := ctx.Value(key).(T)
	if !ok {
		return zero, false
	}
	return v, true
}
