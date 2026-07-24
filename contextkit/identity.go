package contextkit

import "context"

// WithIdentity returns a derived context that carries identity as
// the package-scoped identity value. The type parameter T lets
// callers store any auth-derived struct (claims, principal, merchant
// record, ...) under a single private context key, without the
// package having to know the concrete type.
//
// Calling WithIdentity more than once replaces the previous
// identity on the returned context. The original context is not
// modified.
//
// A nil context is treated as context.Background() so the helper is
// safe to call from places that have not yet prepared a context.
func WithIdentity[T any](ctx context.Context, identity T) context.Context {
	return ctxWithValue(ctx, identityKey{}, identity)
}

// Identity returns the identity value stored on ctx as type T, and a
// bool reporting whether one was present. When no value has been
// stored under the identity key, or the stored value's runtime type
// does not match T, the function returns the zero value of T and
// false. It never panics.
//
// Because the runtime check uses a type assertion, callers must use
// the same T at read time that they used at write time. Reading with
// a different T (for example after a refactor that renames the
// claims type) will surface as a missing value rather than a
// panic — by design, the package favours "no value" over a runtime
// crash on type drift.
func Identity[T any](ctx context.Context) (T, bool) {
	return ctxValue[T](ctx, identityKey{})
}
