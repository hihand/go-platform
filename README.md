# go-platform

A general-purpose Go platform library providing shared infrastructure for backend services. Each package is **stdlib-first**, small, and owns its own wire contract — designed to be imported wholesale or one piece at a time.

## Packages

| Package | Purpose | Doc |
|---------|---------|-----|
| `errkit` | Core error type with code, message, cause, metadata; stdlib `error`-compatible | [docs/errkit.md](docs/errkit.md) |
| `errkit/httperr` | Maps `errkit.Code` → HTTP status codes | [docs/httperr.md](docs/httperr.md) |
| `errkit/grpcerr` | Maps `errkit.Code` → gRPC status codes | [docs/grpcerr.md](docs/grpcerr.md) |
| `logkit` | Structured JSON logger with allocation-conscious hot path | [docs/logkit.md](docs/logkit.md) |
| `contextkit` | Type-safe, stdlib-only request-scoped metadata (`Request` + generic `Identity[T]`) via `context.Context` | [docs/contextkit.md](docs/contextkit.md) |
| `configkit` | YAML + env-var configuration loader on top of Viper, with optional validator | [docs/configkit.md](docs/configkit.md) |
| `responsekit` | Unified HTTP response envelope for Gin, Fiber, net/http | [docs/responsekit.md](docs/responsekit.md) |
| `paginationkit` | Transport-agnostic offset / cursor pagination models | [docs/paginationkit.md](docs/paginationkit.md) |

## Design Goals

- **stdlib-first at the core** — `errkit`, `logkit`, `contextkit`, `responsekit`, `paginationkit` import nothing outside the standard library. The HTTP/gRPC adapters (`errkit/httperr`, `errkit/grpcerr`) and the response adapters (`responsekit/...`) depend on their respective transport library, but never leak it into caller code.
- **`configkit` is the one library-aware package** — it wraps Viper for YAML + env-var loading. Viper is a private implementation detail: no public type, signature, or example mentions it, so swapping the underlying library is a single-file change.
- **stdlib compatible** — implements `error`, `Unwrap()`, works with `errors.Is` / `errors.As`
- **Options-based construction** — single `New(opts ...Option)` entry point for every kit that needs configuration
- **Defensive copies** — metadata is always shallow-copied on write and read
- **No global state** — adapters use explicit `Mapper` instances; no package-level mutability
- **One concern per file** — `spec.go`, `new.go`, `<func>.go`, `common.go`; tests and mocks consolidated (see `RULES.md`)
- **Typed enums** — `Level`, `Key`, `Code` are typed values so typos fail at compile time
- **Wire format owned by the library** — `errkit` defines the wire shape, adapter packages are thin renderers
- **Tolerate nil, return zero on miss** — getters across `contextkit` (and `errkit`'s predicates) never panic on a missing value, a nil receiver, or a type drift

## Quick Start

```bash
go get github.com/hihand/go-platform
```

```go
import (
    "github.com/hihand/go-platform/errkit"
    "github.com/hihand/go-platform/logkit"
    "github.com/hihand/go-platform/responsekit"
)
```

> Each subpackage can be imported individually — pull `errkit` only, or
> `errkit` plus `paginationkit` plus `responsekit`; no implicit coupling.

## Running Examples

```bash
go run main.go
```

`main.go` walks every package: `errkit` construction / wrapping / mapping,
`logkit` levels / context / formatting / caller capture, the
`responsekit` Gin / Fiber / net/http adapters, `configkit` YAML + env + validator
precedence, and `contextkit` request + identity propagation.

## Running Tests

```bash
go test ./...
go test -bench=. ./logkit/     # hot-path allocation benchmarks
```

## Layout

```
errkit/         — core error + sugar constructors
  httperr/      — HTTP status mapping
  grpcerr/      — gRPC status mapping
logkit/         — structured JSON logger
contextkit/     — request-scoped metadata via context.Context
configkit/      — YAML + env-var configuration loader (Viper-backed)
responsekit/    — HTTP response envelope (gin / fiber / nethttp)
paginationkit/  — offset/cursor pagination models
docs/           — per-package guides
main.go         — runnable demo
RULES.md        — file / package conventions
```

See `RULES.md` for the file-naming and file-layout conventions this
project follows.
