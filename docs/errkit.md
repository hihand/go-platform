# errkit

Structured errors with code, message, cause chain, and metadata — compatible with Go's standard `errors` package.

## The Problem

Go's built-in `error` interface is just a string. In a real system you need:

- A **machine-readable code** (not just parsing log text)
- A **cause chain** so `errors.Is` / `errors.As` keep working across service boundaries
- **Structured metadata** for debugging and telemetry
- A clean path to **HTTP / gRPC / GraphQL** without polluting the core package

`errkit` solves all of this.

## Error Interface

```go
type Error interface {
    error                          // standard error interface
    Code() Code                   // machine tag: "NOT_FOUND", "INVALID_ARGUMENT", ...
    Message() string              // human text: "user 42 not found"
    Unwrap() error               // cause chain for errors.Is / errors.As
}
```

## Codes

`errkit.Code` is `string` — no enum, no int, JSON-serializable by default.

```go
const (
    CodeUnknown          Code = "UNKNOWN"
    CodeInvalidArgument  Code = "INVALID_ARGUMENT"
    CodeNotFound         Code = "NOT_FOUND"
    CodeAlreadyExists    Code = "ALREADY_EXISTS"
    CodeUnauthenticated  Code = "UNAUTHENTICATED"
    CodePermissionDenied Code = "PERMISSION_DENIED"
    CodeInternal         Code = "INTERNAL"
    CodeUnavailable      Code = "UNAVAILABLE"
    CodeDeadlineExceeded Code = "DEADLINE_EXCEEDED"
    CodeCanceled         Code = "CANCELED"
)
```

## Construction

### Option-based (canonical)

```go
err := errkit.New(
    errkit.WithCode(errkit.CodeInvalidArgument),
    errkit.WithMessage("field 'email' is required"),
    errkit.WithMetadata(map[string]any{
        "field": "email",
        "value": "",
    }),
)
// => INVALID_ARGUMENT: field 'email' is required
```

Options applied in order; later options override earlier ones for scalar fields. The map in `WithMetadata` is shallow-copied at construction time.

### Sugar constructors

Shortcuts for the most common codes — no need to remember constants.

```go
errkit.NotFound("user 42")
errkit.InvalidArgument("id is required")
errkit.Internal("database unavailable")
errkit.AlreadyExists("email already registered")
errkit.Unauthenticated("token expired")
errkit.PermissionDenied("insufficient access rights")
errkit.Unavailable("service overloaded")
errkit.DeadlineExceeded("upstream timeout")
errkit.Canceled("request cancelled by client")
```

### Zero value

```go
err := errkit.New()
// => UNKNOWN:
```

## Wrapping

`Wrap` attaches errkit attributes to an existing error while preserving the cause chain. Mirrors `fmt.Errorf("%w", err)` semantics — returns `nil` when `err` is `nil`.

```go
cause := errors.New("connection refused")
err := errkit.Wrap(cause,
    errkit.WithCode(errkit.CodeUnavailable),
    errkit.WithMessage("upstream is down"),
)
// => UNAVAILABLE: upstream is down: connection refused

// errors.Is still works
errors.Is(err, cause) // true

// Check the code anywhere in the chain
errkit.IsCode(err, errkit.CodeUnavailable) // true
```

## Predicates

```go
err := errkit.Wrap(cause,
    errkit.WithCode(errkit.CodeUnavailable),
    errkit.WithMessage("upstream is down"),
    errkit.WithMetadata(map[string]any{"retry": true}),
)

errkit.CodeOf(err)        // UNAVAILABLE
errkit.MessageOf(err)    // upstream is down
errkit.MetadataOf(err)   // map[retry:true]

e, ok := errkit.FromError(err) // extracts first errkit.Error in the chain
if ok {
    fmt.Println(e.Code())
}
```

`CodeOf`, `MessageOf`, `MetadataOf`, and `FromError` walk the `Unwrap()` chain automatically using `errors.As`.

## Protocol Adapters

See the full guides:

- [HTTP adapter](httperr.md) — `errkit/httperr`
- [gRPC adapter](grpcerr.md) — `errkit/grpcerr`

## Compatibility with Go's `errors` Package

Every `errkit.Error` is a stdlib `error`, and the cause chain is exposed via `Unwrap()`. This means:

```go
// fmt.Errorf %w still works
wrapped := fmt.Errorf("ctx: %w", err)

// errors.Is walks the chain automatically
errors.Is(wrapped, os.ErrNotExist) // true if any cause matches

// errors.As walks the chain automatically
var e errkit.Error
errors.As(wrapped, &e) // true if any cause is an errkit.Error
```

## File Layout

| File | Purpose |
|------|---------|
| `spec.go` | `Error` interface definition |
| `types.go` | `impl` struct, `config`, `build()` |
| `code.go` | `Code` type, constants, helpers (`CodeOf`, `FromError`, `IsCode`) |
| `new.go` | `New`, `Wrap`, sugar constructors |
| `options.go` | `Option` type and helpers (`WithCode`, `WithMessage`, etc.) |
| `error.go` | `Error()`, `Unwrap()` implementations |
| `message.go` | `MessageOf` helper |
| `metadata.go` | `MetadataAccessor`, `MetadataOf` helper |
| `stack.go` | Stack capture (internal, reserved for future public API) |
