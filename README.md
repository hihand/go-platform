# go-platform

A general-purpose Go platform library providing shared infrastructure for backend services.

## Packages

| Package | Purpose |
|---------|---------|
| `errkit` | Core error type with code, message, cause, and metadata |
| `errkit/httperr` | Maps `errkit.Code` → HTTP status codes |
| `errkit/grpcerr` | Maps `errkit.Code` → gRPC status codes |
| `logkit` | Structured JSON logger with allocation-conscious hot path |
| `responsekit` | Unified HTTP response envelope for Gin, Fiber, net/http |
| `paginationkit` | Transport-agnostic offset/cursor pagination models |

## Design Goals

- **Zero transport dependency** — core packages have no imports outside stdlib
- **stdlib compatible** — implements `error`, `Unwrap()`, works with `errors.Is` / `errors.As`
- **Options-based construction** — single `New(opts ...Option)` entry point
- **Defensive copies** — metadata is always shallow-copied on write and read
- **No global state** — adapters use explicit `Mapper` instances; no package-level mutability

## Running Examples

```bash
go run main.go
```

## Running Tests

```bash
go test ./...
```
