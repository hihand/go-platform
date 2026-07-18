// Package errkit provides a minimal, transport-agnostic error type with code,
// message, cause, and metadata, plus protocol-specific adapters under the
// subpackages httperr, grpcerr, and graphqlerr.
//
// # Design contract
//
//   - The core package has zero dependencies on HTTP, gRPC, or GraphQL types.
//   - Errors implement the standard error interface and are compatible with
//     errors.Is / errors.As / Unwrap.
//   - Construction is options-based; there is a single New(opts ...Option)
//     entry point that produces an Error.
//   - A small set of sugar constructors (NotFound, InvalidArgument, Internal,
//     ...) wrap the common cases.
//   - Metadata is a JSON-friendly map[string]any and is defensive-copied on
//     both ingress and access.
//   - Stack traces are recorded when WithStack is supplied but are reserved
//     for a future stable API and are not yet exposed publicly.
//   - Cause is exposed exclusively through Unwrap(); there is no separate
//     Cause() method on the interface.
//
// # File layout
//
// One concern per file. The Error interface lives in spec.go; the storage
// type and private constructor live in types.go; the public constructors live
// in new.go; option helpers live in options.go; stack capture lives in
// stack.go; and code/message/metadata accessors live in their own files.
package errkit

// Error is the canonical interface implemented by every error produced by
// this package and by any adapter that wants to round-trip with it.
//
// Implementations must also implement MetadataAccessor (defined in
// metadata.go) when they carry metadata, although helpers in this package
// tolerate the absence of Metadata for non-errkit errors.
type Error interface {
	error
	// Code returns the stable, machine-friendly identifier for this error.
	Code() Code
	// Message returns the human-readable message.
	Message() string
	// Unwrap returns the underlying cause, or nil if there is none.
	// It is the single source of truth for the cause chain and is used
	// by errors.Is and errors.As.
	Unwrap() error
}
