# logkit

Structured (JSON) logger for the go-platform ecosystem. Stdlib-only, allocation-conscious hot path, single seam for context-driven correlation.

## At a glance

```go
logger := logkit.New(
    logkit.WithOutput(os.Stdout),
    logkit.WithService("payment-api", "1.2.0"),
    logkit.WithDeployment("production"),
    logkit.WithContextMapper(myApp.ExtractLogAttrs),
)

logger.Info("payment created",
    logkit.String(logkit.KeyEvent, "payment.created"),
    logkit.String(logkit.AnyKey("payment.id"), "pay-001"),
    logkit.Int64(logkit.AnyKey("payment.amount"), 100),
)
// 2026-07-22T14:00:00Z INFO payment created event=payment.created service.name=payment-api service.version=1.2.0 deployment.environment=production payment.id=pay-001 payment.amount=100
```

## The Problem

`log/slog` exists, and so do `zap`, `zerolog`, `logrus`. logkit is **not** trying
to replace them — it is a deliberately small replacement that fits the rest
of `go-platform`:

- **stdlib only.** The core package never imports third-party libraries, so a
  service that already pulls `gin-gonic/gin`, `gofiber/fiber/v2`, or
  `google.golang.org/grpc` does not need any new top-level dependency.
- **Typed enums for schema fields.** `Level` and `Key` are typed values, so
  typos (`Level.Infoo`, `Key("even t")`) fail at compile time.
- **Single context seam.** The logger is **context-unaware**. The
  application — or a separate package — owns a `ContextMapper` that reads
  what *it* stashed in `context.Context` (`request.id`, `trace.id`, user
  claims, …) and returns a slice of `Attr`. Everything else becomes the
  logger's static + `With` state.
- **Allocation-conscious hot path.** A `sync.Pool`-backed scratch buffer and
  a hand-rolled JSON encoder keep the per-call cost low; `Debugf` skips
  `Sprintf` entirely when the level is gated.

## Levels

```go
const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
)
```

`WithMinLevel(...)` gates everything below. Unknown values are clamped to
`LevelInfo` so misconfiguration never panics.

```go
logger := logkit.New(logkit.WithMinLevel(logkit.LevelWarn))
logger.Debug("dropped")  // skipped
logger.Info("skipped")   // skipped
logger.Warn("kept")      // emitted
logger.Error("kept")     // emitted
```

## Keys

Canonical schema fields are typed constants. Application-defined fields use
the `AnyKey` escape hatch.

```go
logkit.KeyEvent                  // "event"
logkit.KeyTimestamp              // "timestamp"
logkit.KeyLevel                  // "level"
logkit.KeyMessage                // "message"
logkit.KeyCaller                 // "caller"
logkit.KeyServiceName            // "service.name"
logkit.KeyServiceVersion         // "service.version"
logkit.KeyDeploymentEnvironment  // "deployment.environment"

logkit.AnyKey("payment.id")      // typed Key created at call site
```

`KeyEvent` is special: when present on the record (call-site attr or
`With` attr), it is **lifted to a top-level `event` field** and removed from
the free-form attributes. This makes log queries (`event: payment.failed`)
cheap without forcing every schema to match the same field name.

## Attr

`Attr` is the typed key/value pair passed to every log method:

| Constructor | Notes |
|-------------|-------|
| `String(key, value)` | Hot path — most business fields are strings. |
| `Int(key, value)` | Stored as `int64` internally. |
| `Int64(key, value)` | Same. |
| `Uint64(key, value)` | — |
| `Float64(key, value)` | IEEE-754 bits stored so the encoder can hand them straight to `strconv.AppendFloat`. |
| `Bool(key, value)` | — |
| `Dur(key, time.Duration)` | Stored as int64 nanoseconds. |
| `Time(key, time.Time)` | Rendered with `time.AppendFormat` (no intermediate string). |

## Logger API

```go
type Logger interface {
    // Plain (no context)
    Debug(msg string, attrs ...Attr)
    Info(msg string, attrs ...Attr)
    Warn(msg string, attrs ...Attr)
    Error(msg string, attrs ...Attr)

    // *Context — invokes the configured ContextMapper on ctx (if any)
    DebugContext(ctx context.Context, msg string, attrs ...Attr)
    InfoContext(ctx context.Context, msg string, attrs ...Attr)
    WarnContext(ctx context.Context, msg string, attrs ...Attr)
    ErrorContext(ctx context.Context, msg string, attrs ...Attr)

    // Formatted — fmt.Sprintf semantics; Sprintf is skipped when the level
    // is disabled, so Debugf costs a single variadic slice header.
    Debugf(format string, args ...any)
    Infof(format string, args ...any)
    Warnf(format string, args ...any)
    Errorf(format string, args ...any)

    // Formatted + *Context
    DebugContextf(ctx context.Context, format string, args ...any)
    InfoContextf(ctx context.Context, format string, args ...any)
    WarnContextf(ctx context.Context, format string, args ...any)
    ErrorContextf(ctx context.Context, format string, args ...any)

    // With returns a child logger with attrs permanently attached.
    // Child attrs override parent attrs on duplicate keys.
    With(attrs ...Attr) Logger
}
```

### Precedence

For any record the merged attribute order is (later wins on duplicate keys):

```
static → withAttrs → ctxAttrs → callAttrs
```

`withAttrs` is the slice a `With(...)` chain has built up. `ctxAttrs` is
the slice the configured `ContextMapper` returned for the supplied
context. `callAttrs` are the attrs the caller passed to the current
`Info(...)` / `ErrorContext(...)` / … call. A typed enum encoding means
key comparison is just a string-equality check on the underlying value.

## Options

All options are functional. They are applied in order; later options
override earlier ones for scalar fields.

| Option | Default | Purpose |
|--------|---------|---------|
| `WithOutput(w io.Writer)` | `os.Stdout` | Sink. The logger does not own the writer. |
| `WithMinLevel(Level)` | `LevelInfo` | Drop anything below this level. |
| `WithService(name, version string)` | empty | Static `service.name` + `service.version`. |
| `WithDeployment(env string)` | empty | Static `deployment.environment`. |
| `WithStatic(key Key, value string)` | — | Free-form static `key:value` (cluster, region, build SHA). |
| `WithCaller()` | off | Capture caller as `filepath:line`. Internal logkit frames are skipped so the user sees the real call site. |
| `WithContextMapper(mapper ContextMapper)` | nil | The context seam — see below. |

```go
logger := logkit.New(
    logkit.WithOutput(os.Stdout),
    logkit.WithService("payment-api", "1.2.0"),
    logkit.WithDeployment("production"),
    logkit.WithCaller(),
    logkit.WithStatic(logkit.AnyKey("region"), "ap-southeast-1"),
)
```

### Caller capture

`WithCaller()` is off by default to keep the disabled `Debug` path zero-cost.
When enabled, every emitted record carries a single `"caller":"dir/file.go:line"`
field. The file path is trimmed to its last two segments, so the field stays
grep-friendly without giving up enough context to navigate.

## ContextMapper

```go
// ContextMapper receives ctx and an optional scratch slice, and returns
// the slice grown with whatever attrs it wants emitted on this record.
// Returning the input unchanged is the no-op signal.
type ContextMapper func(ctx context.Context, attrs []Attr) []Attr

logger := logkit.New(
    logkit.WithContextMapper(func(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
        if v, ok := ctx.Value(reqIDKey).(string); ok && v != "" {
            return append(attrs, logkit.String(logkit.AnyKey("request.id"), v))
        }
        return attrs
    }),
)
```

Why a function instead of an interface?

- Functions carry no inherent state. Composition (chaining multiple
  mappers) is a tiny wrapper at the call site.
- The append-style signature reuses the scratch slice — no per-call slice
  allocation.

If no mapper is configured, **the context is ignored entirely** and the
non-context methods stay zero-allocation. The level gate still fires.

## `With` — derived loggers

```go
scoped := logger.With(
    logkit.String(logkit.AnyKey("request.id"), "req-with"),
)
scoped.Info("scoped event")
```

- Returns a new `Logger`; the original is untouched.
- Duplicate keys collide in favour of the later (child) attribute.
- `With()` with no args returns the same logger.

## Wire format

One JSON object per record, one line per record:

```json
{"timestamp":"2026-07-22T14:00:00.000000000Z","level":"INFO","message":"payment created","event":"payment.created","service.name":"payment-api","service.version":"1.2.0","deployment.environment":"production","payment.id":"pay-001","payment.amount":100}
```

Field order is **schema fields first** (`timestamp`, `level`, `message`,
optional `event`), then `static` (in `WithService` / `WithDeployment` /
`WithStatic` order), then merged attrs, then optional `caller`. The
encoder is hand-written (no `encoding/json`) so it can write straight
into the pooled scratch buffer.

Strings are escaped only for the characters JSON requires (`"`, `\`, `\n`,
`\r`, `\t`, and control bytes `< 0x20`). UTF-8 is not validated — the
caller owns input.

## Composition with errkit

`logkit` does not depend on `errkit`, so error attributes must be hoisted
manually at the call site:

```go
err := errkit.NotFound("user 42")
logger.ErrorContext(ctx, "lookup failed",
    logkit.String(logkit.KeyEvent, "user.not_found"),
    logkit.String(logkit.AnyKey("errkit.code"), string(errkit.CodeOf(err))),
    logkit.String(logkit.AnyKey("errkit.message"), errkit.MessageOf(err)),
)
```

A future package (or your own 3-line helper) can wrap this so the
boilerplate lives in one place; `logkit` itself stays stdlib-only.

## Performance

`logkit/logkit_bench_test.go` pins the hot path:

| Bench | Notes |
|-------|-------|
| `BenchmarkHotPath_NoAttrs` | Plain `Info("ok")`. |
| `BenchmarkHotPath_String` | Two string attrs. |
| `BenchmarkHotPath_Mixed` | Pre-built attr slice + `ErrorContext`. |
| `BenchmarkHotPath_PerCall_WithCaller` | Caller capture on every call (always-allocated path). |
| `BenchmarkHotPath_TimeAndDur` | `Time` + `Dur` encode via direct `AppendFormat`. |
| `BenchmarkHotPath_LoggerWithChainedAttrs` | Three-level `.With(...)` chain. |

`WithCaller()` and the level gate both branch on the hot path but the
encoder writes into the pooled buffer with no interface boxing.
`WithCaller()` is recommended off by default — flip it on for `WARN`/`ERROR`
sinks if you want a permanent pointer to the line that emitted the record.

## File Layout

| File | Purpose |
|------|---------|
| `spec.go` | `Logger` interface (4 levels × 4 forms + `With`) |
| `enum.go` | `Level`, `Key` typed enums + canonical names |
| `new.go` | `impl` struct + `New(opts ...Option)` |
| `options.go` | Functional options (`WithOutput`, `WithMinLevel`, `WithService`, `WithDeployment`, `WithStatic`, `WithCaller`) |
| `mapper.go` | `ContextMapper` type + `WithContextMapper` |
| `attr.go` | `Attr` union + typed constructors |
| `log.go` | Hot path (level gate → merge → encode) |
| `record.go` | Internal value passed to the encoder |
| `encode.go` | Hand-written JSON encoder + escaping |
| `caller.go` | `runtime.CallersFrames` capture + path trim |
| `with.go` | `Logger.With` derivation |
| `*_test.go` | All tests in one file per package |
| `logkit_bench_test.go` | Hot-path benchmarks |
| `logkit_mock_test.go` | Shared test mocks (capture buffer + reqID helpers) |
| `example_test.go` | Runnable godoc examples |
