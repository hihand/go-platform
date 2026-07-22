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
    error                 // standard error interface
    Code() Code           // machine tag: "NOT_FOUND", "INVALID_ARGUMENT", ...
    Message() string      // human text: "user 42 not found"
    Unwrap() error        // cause chain for errors.Is / errors.As
}
```

## Codes

`errkit.Code` is `string` — no enum, no int, JSON-serializable by default.

The constants are grouped by lifecycle stage so 4xx codes live together, 5xx
codes live together, and protocol-time exceptions (`Canceled`,
`DeadlineExceeded`, `RequestTimeout`) sit at the top. Wire labels follow
**gRPC conventions** rather than HTTP-internal names — `CodeInternal` is
`"INTERNAL"`, not `"INTERNAL_SERVER_ERROR"` — so the same string round-trips
through both `httperr` and `grpcerr`.

### Transport / lifecycle

| Code | Wire |
|------|------|
| `CodeUnknown` | `"UNKNOWN"` |
| `CodeCanceled` | `"CANCELED"` |
| `CodeDeadlineExceeded` | `"DEADLINE_EXCEEDED"` |
| `CodeRequestTimeout` | `"REQUEST_TIMEOUT"` |

### Client errors (4xx)

| Code | Wire |
|------|------|
| `CodeInvalidArgument` | `"INVALID_ARGUMENT"` |
| `CodeUnauthenticated` | `"UNAUTHENTICATED"` |
| `CodePermissionDenied` | `"PERMISSION_DENIED"` |
| `CodeNotFound` | `"NOT_FOUND"` |
| `CodeConflict` | `"CONFLICT"` |
| `CodeAlreadyExists` | `"ALREADY_EXISTS"` |
| `CodeDuplicate` | `"DUPLICATE"` |
| `CodeMethodNotAllowed` | `"METHOD_NOT_ALLOWED"` |
| `CodeNotAcceptable` | `"NOT_ACCEPTABLE"` |
| `CodeGone` | `"GONE"` |
| `CodeLengthRequired` | `"LENGTH_REQUIRED"` |
| `CodePreconditionFailed` | `"PRECONDITION_FAILED"` |
| `CodePayloadTooLarge` | `"PAYLOAD_TOO_LARGE"` |
| `CodeURITooLong` | `"URI_TOO_LONG"` |
| `CodeUnsupportedMediaType` | `"UNSUPPORTED_MEDIA_TYPE"` |
| `CodeRangeNotSatisfiable` | `"RANGE_NOT_SATISFIABLE"` |
| `CodeExpectationFailed` | `"EXPECTATION_FAILED"` |
| `CodeMisdirectedRequest` | `"MISDIRECTED_REQUEST"` |
| `CodeUnprocessableEntity` | `"UNPROCESSABLE_ENTITY"` |
| `CodeLocked` | `"LOCKED"` |
| `CodeFailedDependency` | `"FAILED_DEPENDENCY"` |
| `CodeTooManyRequests` | `"TOO_MANY_REQUESTS"` |
| `CodeRequestHeaderFieldsTooLarge` | `"REQUEST_HEADER_FIELDS_TOO_LARGE"` |
| `CodeUnavailableForLegalReasons` | `"UNAVAILABLE_FOR_LEGAL_REASONS"` |
| `CodePaymentRequired` | `"PAYMENT_REQUIRED"` |
| `CodeUpgradeRequired` | `"UPGRADE_REQUIRED"` |

### Server errors (5xx)

| Code | Wire |
|------|------|
| `CodeInternal` | `"INTERNAL"` |
| `CodeNotImplemented` | `"NOT_IMPLEMENTED"` |
| `CodeBadGateway` | `"BAD_GATEWAY"` |
| `CodeUnavailable` | `"UNAVAILABLE"` |
| `CodeDataLoss` | `"DATA_LOSS"` |
| `CodeNetworkAuthenticationRequired` | `"NETWORK_AUTHENTICATION_REQUIRED"` |

### Custom codes

Declare your own when you need a domain-specific wire label:

```go
const errkit.CodePaymentRequired errkit.Code = "PAYMENT_REQUIRED"
```

The default adapters ignore unknown codes (`CodeUnknown` and `Code("…")` both
fall back to 500 / `Unknown`), so pick a label with `NewMapper` whenever the
default mapping would be wrong.

### Picking the right conflict-style code

Three 409-family codes coexist on purpose. Pick the narrowest one:

- **`CodeConflict`** — generic business-rule clash. "You can't transition
  a published invoice back to draft."
- **`CodeAlreadyExists`** — gRPC-flavoured: another writer created the
  resource while this request was in flight.
- **`CodeDuplicate`** — strictly stronger: a uniqueness constraint (DB
  unique index, idempotency key) was violated.

`httperr` and `grpcerr` map all three to 409 / `Aborted` by default. Wire
`CodeDuplicate` to a more specific status via `NewMapper` if you have to.

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
errkit.Conflict("optimistic lock failed")
errkit.Unauthenticated("token expired")
errkit.PermissionDenied("insufficient access rights")
errkit.Unavailable("service overloaded")
errkit.DeadlineExceeded("upstream timeout")
errkit.RequestTimeout("client gave up")
errkit.TooManyRequests("rate limit exceeded")
errkit.UnprocessableEntity("business rule rejected the request")
errkit.PayloadTooLarge("file too big")
errkit.MethodNotAllowed("use PUT")
errkit.NotAcceptable("no JSON variant available")
errkit.Gone("resource was archived")
errkit.PreconditionFailed("ETag mismatch")
errkit.UnsupportedMediaType("only application/json")
errkit.BadGateway("upstream returned an invalid response")
errkit.NotImplemented("feature is on the roadmap")
errkit.DataLoss("read after write found torn pages")
errkit.Canceled("request cancelled by client")
```

> Codes that intentionally have no sugar constructor — `Duplicate`,
> `PaymentRequired`, `UpgradeRequired`, `URITooLong`, `MisdirectedRequest`,
> `Locked`, `FailedDependency`, `RangeNotSatisfiable`, `ExpectationFailed`,
> `RequestHeaderFieldsTooLarge`, `UnavailableForLegalReasons`,
> `LengthRequired`, `NetworkAuthenticationRequired` — are reachable via
> `New(WithCode(...), WithMessage(...))`. The deliberate gap forces callers
> to pick a wire policy rather than re-typing the constant.

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
errkit.MessageOf(err)     // upstream is down
errkit.MetadataOf(err)    // map[retry:true]

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

## Sibling packages

- [logkit](logkit.md) — structured JSON logger
- [responsekit](responsekit.md) — unified HTTP response envelope for Gin, Fiber, net/http
- [paginationkit](paginationkit.md) — offset / cursor pagination models

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
| `code.go` | `Code` type, constants, helpers (`CodeOf`, `FromError`, `IsCode`) |
| `new.go` | `New`, `Wrap`, sugar constructors |
| `options.go` | `Option` type and helpers (`WithCode`, `WithMessage`, etc.) |
| `error.go` | `Error()`, `Unwrap()` implementations |
| `message.go` | `MessageOf` helper |
| `metadata.go` | `MetadataAccessor`, `MetadataOf` helper |
| `stack.go` | Stack capture (internal, reserved for future public API) |
