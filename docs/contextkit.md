# contextkit

A lightweight, type-safe convention for sharing request-scoped metadata through `context.Context`. Stdlib only — no logging, tracing, OpenTelemetry, or middleware.

## At a glance

```go
// Transport edge stamps the request id once.
ctx := contextkit.WithRequest(ctx, contextkit.Request{
    RequestID: "req-7f3a",
    TraceID:   "trace-001",
    SpanID:    "span-002",
})

// Downstream layers read it back without depending on a transport type.
req := contextkit.GetRequest(ctx)
// req.RequestID == "req-7f3a"

// Identity is generic over any auth-shaped struct. The package has no
// opinion on the shape.
type Claims struct {
    UserID     string
    MerchantID string
}
ctx = contextkit.WithIdentity(ctx, Claims{UserID: "u-42"})

c, ok := contextkit.Identity[Claims](ctx)
// ok == true, c.UserID == "u-42"
```

## Design contract

- **stdlib only.** No dependency on HTTP, gRPC, tracing, OpenTelemetry, or any third-party framework.
- **Private context keys.** Values are stored behind unexported `struct{}` keys so callers cannot accidentally collide with packages that use string keys. The key types are deliberately not exported.
- **All getters tolerate a nil context.** They never panic, never dereference a nil receiver.
- **All getters return the zero value on miss.** `GetRequest` returns the zero `Request{}`; `Identity[T]` returns `(zero T, false)`.
- **Type drift is reported as missing, not as a panic.** Reading `Identity[T]` with the wrong `T` after a refactor surfaces as `(zero, false)`.
- **No global state, no mutex, no reflection at runtime.** The package owns nothing beyond storage conventions.
- **No logger, no auth, no middleware, no trace/span wrappers.** Logging and tracing systems read the metadata; this package does not depend on them.

## When to use it

| Use `contextkit` when                                                                                         | Don't use it when                                                                |
|----------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------|
| Multiple layers (HTTP handler → service → repo) need the same request-scoped value.                            | You only need a single value across one function — pass it as a parameter.       |
| The shape of the value is application-owned (claims, principal, merchant record, request id).                  | The value is a global config flag — promote it to `configkit` / `envvar`.        |
| You want a typed, refactor-safe replacement for `ctx.Value("user.id").(string)` patterns.                      | You need OTel-style span control — `contextkit` is plain strings, not span APIs. |
| You want the same getter to work across HTTP, gRPC, queue workers, and CLI runners.                           | You need cross-cutting behaviour (logging, metrics) — that's `logkit` / OTel.    |

## Public API

The package ships exactly four functions and one struct. No constructors, no options, no sugar — a utility this small should be one screen of godoc.

```go
type Request struct {
    RequestID string
    TraceID   string
    SpanID    string
}

func WithRequest(ctx context.Context, req Request) context.Context
func GetRequest(ctx context.Context) Request

func WithIdentity[T any](ctx context.Context, identity T) context.Context
func Identity[T any](ctx context.Context) (T, bool)
```

### `Request`

A plain struct with three optional `string` fields. There is no `RequestID` validation, no UUID parsing, no tracing constants — those concerns belong to the tracing layer that *fills* the struct.

```go
ctx = contextkit.WithRequest(ctx, contextkit.Request{
    RequestID: "req-7f3a",
    TraceID:   "trace-001",
    SpanID:    "span-002",
})
```

- `WithRequest` overwrites the previous `Request` on the returned context. The original context is unchanged.
- `GetRequest` returns a copy. Mutating the result has no effect on what other layers see.
- `GetRequest(nil)` returns the zero `Request{}`. It never panics.
- `GetRequest(ctx)` on an empty context returns the zero `Request{}`. It never panics.

> **Naming note:** the getter is `GetRequest`, not `Request`, because Go forbids a function and a struct type from sharing a name in the same package. The `Get` prefix keeps the call site readable.

### `Identity[T]`

A generic carrier for one request-scoped struct. The package has no opinion on the struct shape — `Claims`, `Principal`, `MerchantRecord`, `Session`, anything goes. Each call site picks its own type and uses the same `T` at read time.

```go
type Claims struct {
    UserID     string
    MerchantID string
    Roles      []string
}

ctx = contextkit.WithIdentity(ctx, Claims{
    UserID:     "u-42",
    MerchantID: "m-7",
})

c, ok := contextkit.Identity[Claims](ctx)
if !ok {
    // no identity on this context (e.g. anonymous endpoint, test, queue worker)
}
_ = c.UserID
```

#### Reading with the wrong `T`

Refactors that rename or move the struct are safe by default: the read-side `Identity[T]` does a runtime type assertion, and a mismatch surfaces as `(zero, false)`, not as a panic. Use the same `T` at read time that you used at write time — if you can't, the value was written by a different code path and the caller should treat it as absent.

```go
// Stored as Claims...
ctx = contextkit.WithIdentity(ctx, Claims{UserID: "u-42"})

// ...asking for a different struct returns (zero, false).
got, ok := contextkit.Identity[OtherClaims](ctx)
// ok == false, got == OtherClaims{}
```

#### Pointer payloads

Storing and reading through a pointer type works and round-trips the exact pointer. Asking for the value type on a pointer payload (or vice-versa) is also a `(zero, false)` — no implicit dereferencing.

```go
c := &Claims{UserID: "u-42"}
ctx = contextkit.WithIdentity(ctx, c)

got, ok := contextkit.Identity[*Claims](ctx)
// got == c (same pointer)
_, ok = contextkit.Identity[Claims](ctx)
// ok == false — wrong T
```

## Behaviour reference

| Call                                          | Result                                              |
|-----------------------------------------------|-----------------------------------------------------|
| `WithRequest(ctx, req)` with `ctx == nil`     | Returns a context rooted at `context.Background()`.  |
| `WithRequest(ctx, req)` on top of a previous  | Overwrites on the returned context; parent untouched. |
| `GetRequest(ctx)` with `ctx == nil`           | Returns the zero `Request{}`.                        |
| `GetRequest(ctx)` on empty context            | Returns the zero `Request{}`.                        |
| `WithIdentity[T](ctx, v)` with `ctx == nil`   | Returns a context rooted at `context.Background()`.  |
| `Identity[T](ctx)` with `ctx == nil`          | Returns `(zero T, false)`.                           |
| `Identity[T](ctx)` on empty context           | Returns `(zero T, false)`.                           |
| `Identity[T](ctx)` after `WithIdentity[U]`    | Returns `(zero T, false)`.                           |
| `Identity[T](ctx)` after `WithIdentity[T]`    | Returns `(v, true)`.                                 |

## Patterns

### Transport edge → downstream service → repo

```go
// HTTP handler — the only place that knows about headers.
func (h *Handler) GetUser(c *gin.Context) {
    ctx := contextkit.WithRequest(c.Request.Context(), contextkit.Request{
        RequestID: c.GetHeader("X-Request-ID"),
    })
    ctx = contextkit.WithIdentity(ctx, claimsFromJWT(c))

    user, err := h.svc.GetUser(ctx, id)
    // ...
}

// Service — depends on contextkit, not on gin.
func (s *Svc) GetUser(ctx context.Context, id string) (*User, error) {
    req := contextkit.GetRequest(ctx)
    c, ok := contextkit.Identity[Claims](ctx)
    _ = req
    _ = c
    _ = ok
    // ...
}
```

The transport can be swapped (HTTP → gRPC → queue worker) without the service or repo changing: they all read the same `contextkit` getters.

### Pairing with `logkit`

The package does not depend on `logkit`, but `logkit.WithContextMapper` is the natural place to lift request metadata into log records:

```go
logger := logkit.New(
    logkit.WithService("payment-api", "1.2.0"),
    logkit.WithContextMapper(func(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
        req := contextkit.GetRequest(ctx)
        if req.RequestID != "" {
            attrs = append(attrs, logkit.String("request.id", req.RequestID))
        }
        if c, ok := contextkit.Identity[Claims](ctx); ok && c.MerchantID != "" {
            attrs = append(attrs, logkit.String("merchant.id", c.MerchantID))
        }
        return attrs
    }),
)

logger.InfoContext(ctx, "payment created") // record now carries request.id + merchant.id
```

### Anonymous-default pattern

Read-side code can safely default to anonymous behaviour without remembering whether middleware ran:

```go
func requireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        c, err := parseJWT(r)
        if err != nil {
            http.Error(w, "unauthenticated", http.StatusUnauthorized)
            return
        }
        ctx := contextkit.WithIdentity(r.Context(), c)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func handlePayment(w http.ResponseWriter, r *http.Request) {
    c, ok := contextkit.Identity[Claims](r.Context())
    if !ok {
        // anonymous path — falls back to public pricing
        servePublicPricing(w)
        return
    }
    servePrivatePricing(w, c)
}
```

## Anti-patterns

The package is intentionally tiny, and the boundary between "contextkit should do this" and "the caller should do this" is deliberate. Avoid:

- **Don't store a logger in the context.** Use `logkit.New(...)` + `logkit.WithContextMapper` to lift values out of the context instead.
- **Don't store tracing or span *objects* in the context.** The package only carries plain strings — let the tracing system own span lifetime.
- **Don't store configuration values.** They don't vary per request, and the indirection through `context.Context` is the wrong tool. Use `configkit` or plain constructor arguments.
- **Don't use string context keys.** The package deliberately hides keys as private `struct{}` types; reuse the public API instead of building your own.

## File Layout

| File                     | Purpose                                                                                |
|--------------------------|----------------------------------------------------------------------------------------|
| `spec.go`                | Package doc comment; design contract + file layout                                     |
| `common.go`              | Private keys (`requestKey`, `identityKey`) + shared `ctxWithValue` / `ctxValue[T]` helpers |
| `request.go`             | `Request` struct + `WithRequest` / `GetRequest`                                          |
| `identity.go`            | Generic `WithIdentity[T]` / `Identity[T]`                                                |
| `contextkit_test.go`     | All package tests (13 tests, table-driven + parallel + race-clean)                       |
| `example_test.go`        | Runnable godoc examples for `WithRequest`, `WithIdentity`, `Identity`                    |
