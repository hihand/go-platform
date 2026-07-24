// Package contextkit provides a lightweight, type-safe convention for
// sharing request-scoped metadata through context.Context.
//
// # Design contract
//
//   - The core package depends only on the Go standard library. There is
//     no dependency on HTTP, gRPC, tracing, OpenTelemetry, logging, or
//     any third-party framework.
//   - Values are stored behind private context keys so callers cannot
//     accidentally collide with packages that use string keys. The key
//     types are deliberately unexported.
//   - All getters tolerate a nil context and a missing value. When the
//     requested value is not in the context they return the zero value
//     (and, for typed getters, a bool reporting absence).
//   - The package owns only storage conventions. It does NOT provide
//     loggers, authentication helpers, middleware, trace/span wrappers,
//     or any "request scope" abstraction beyond plain metadata fields.
//   - There is no global state, no mutex, and no reflection at runtime.
//
// # File layout
//
// One concern per file. The package documentation lives in spec.go;
// the private context keys and shared helpers live in common.go; the
// request-metadata API lives in request.go; and the generic identity
// API lives in identity.go.
package contextkit
