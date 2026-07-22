# responsekit

A small, framework-agnostic HTTP response helper for the go-platform ecosystem. One envelope, three adapters (Gin, Fiber, `net/http`), byte-identical wire output.

## At a glance

```go
// Gin
func getUser(c *gin.Context) {
    u, err := userService.Get(c, id)
    if err != nil {
        responsekit.GinError(c, err)
        return
    }
    responsekit.GinOK(c, u)
}

// Fiber
func getUser(c *fiber.Ctx) error {
    u, err := userService.Get(c.UserContext(), id)
    if err != nil {
        return responsekit.FiberError(c, err)
    }
    return responsekit.FiberOK(c, u)
}

// net/http
func getUser(w http.ResponseWriter, r *http.Request) {
    u, err := userService.Get(r.Context(), id)
    if err != nil {
        responsekit.HTTPError(w, r, err)
        return
    }
    responsekit.HTTPOK(w, r, u)
}
```

Same wire output from all three:

```json
{"data":{"id":"u-1","name":"alice"}}
{"error":{"code":"NOT_FOUND","message":"user 42 not found"}}
```

## Wire format

responsekit owns the response shape; the adapters are thin renderers.

### Success

```json
{"data": <any>}
```

`data` is intentionally typed as `any`. A nil `data` renders as `null` (not
omitted) so the wire shape stays predictable.

```json
{"data": null}
```

### Error

```json
{"error": {"code": "NOT_FOUND", "message": "user 42 not found"}}
```

- `code` is the stable, machine-readable `errkit.Code` string — so a client
  can branch on `code` without parsing the human message.
- `message` is the human-readable description, derived from
  `errkit.MessageOf(err)`.

`metadata` from `errkit` is intentionally **not** included in the default
wire shape:

- Production clients rarely care, and exposing arbitrary `map[string]any`
  is a leak surface.
- If you do want to leak it, use the `*JSON` passthrough adapter and build
  the body yourself.

What you will **not** find in either envelope:

- success/status flags
- timestamps
- request IDs / trace IDs
- pagination metadata
- the original request path

Those concerns live in middleware (request-id propagation, timing, audit)
and in structured logging, not in response bodies.

## Adapters

All three adapters wrap the same three helpers in `common.go` so the
output is identical byte-for-byte. The `*JSON` passthrough is the single
escape hatch for non-standard status / body shapes — the caller is then
responsible for matching the wire format if they want platform consistency.

### Gin — `gin.go`

```go
func GinOK(c *gin.Context, data any)
func GinCreated(c *gin.Context, data any)
func GinAccepted(c *gin.Context, data any)
func GinNoContent(c *gin.Context)
func GinError(c *gin.Context, err error)
func GinJSON(c *gin.Context, status int, body any)   // passthrough
```

`GinNoContent` calls `c.Writer.WriteHeaderNow()` after `c.Status(204)` so
the 204 is on the wire immediately — Gin's writer only flushes on the
first body write, and a 204 has no body. Skipping `WriteHeaderNow` would
queue the 204 until the next middleware runs and cause spurious "Content-Length
missing" warnings on some clients.

### Fiber — `fiber.go`

```go
func FiberOK(c *fiber.Ctx, data any) error
func FiberCreated(c *fiber.Ctx, data any) error
func FiberAccepted(c *fiber.Ctx, data any) error
func FiberNoContent(c *fiber.Ctx) error
func FiberError(c *fiber.Ctx, err error) error
func FiberJSON(c *fiber.Ctx, status int, body any) error   // passthrough
```

Fiber's contract requires handlers to return an `error`, so each helper
returns the result of the underlying `c.JSON(...)` / `c.SendStatus(...)`.
Handlers stay a single `return`.

### `net/http` — `nethttp.go`

```go
func HTTPOK(w http.ResponseWriter, r *http.Request, data any)
func HTTPCreated(w http.ResponseWriter, r *http.Request, data any)
func HTTPAccepted(w http.ResponseWriter, r *http.Request, data any)
func HTTPNoContent(w http.ResponseWriter, r *http.Request)
func HTTPError(w http.ResponseWriter, r *http.Request, err error)
func HTTPJSON(w http.ResponseWriter, r *http.Request, status int, body any)   // passthrough
```

`net/http` does not ship a JSON shortcut, so `nethttp.go` owns its own
`writeJSON` (see "Marshal behaviour" below). The `_ *http.Request`
parameter on every helper is a deliberate shape match with Gin and Fiber
even though `nethttp.go` does not use it — it lets you swap adapters
without changing handler signatures, and a future logging / context-derived
envelope hook can land in one place without breaking callers.

## Error → status code

`GinError` / `FiberError` / `HTTPError` derive the HTTP status from the
error via `errkit/httperr`. The mapping is exhaustive for every built-in
`errkit.Code`; unknown codes fall back to 500. See
[docs/httperr.md](./httperr.md) for the table.

```go
err := errkit.NotFound("user 42")
responsekit.GinError(c, err)
// 404 Not Found
// {"error":{"code":"NOT_FOUND","message":"user 42 not found"}}

err := errkit.InvalidArgument("id is required")
responsekit.GinError(c, err)
// 400 Bad Request
// {"error":{"code":"INVALID_ARGUMENT","message":"id is required"}}
```

For a custom rule (e.g. remap `CodeNotFound → 200`) construct your own
`errkit/httperr.Mapper` and call it yourself — `responsekit`'s adapters
take the lazy path of `httperr.StatusCode(err)`.

## Error → wire code

`errorEnvelope` in `common.go` extracts the public code from any `error`:

| Input | Output `code` |
|-------|---------------|
| `nil` | `"INTERNAL"` |
| `errkit.Error` (anywhere in the cause chain) | its `.Code()` |
| anything else | `"INTERNAL"` |

The `INTERNAL` fallback for plain errors is intentional: a raw Go error
reaching the wire is treated as a server-side bug, so the public label is
the generic catch-all while the original `err.Error()` is preserved in
`message` for the client to log.

## Marshal behaviour

- **Gin** and **Fiber** use the framework's own JSON writer. output is
  `application/json`-tagged by the framework.
- **`net/http`** uses `json.Marshal` (not `json.NewEncoder`) so the body
  has **no trailing newline**. That matches Gin / Fiber byte-for-byte
  — `httptest` equality works across adapters.

  Marshal failures fall back to a 500 plain-text `"internal error"`. By
  the time Marshal can fail, the headers might already be on the wire
  for a non-trivial chunk of state, so the fallback is a safe degradation
  rather than an attempt at recovery.

## Pairing with errkit

The pairs below are the patterns production services use. There is nothing
forcing them — responsekit does not import errkit at the framework boundary,
so the wiring is composition, not configuration.

### Handler that may return a typed error

```go
func (s *UserService) Get(ctx context.Context, id string) (*User, error) {
    u, err := s.repo.Find(ctx, id)
    if errors.Is(err, repo.ErrNotFound) {
        return nil, errkit.NotFound("user " + id)
    }
    if err != nil {
        return nil, errkit.Wrap(err, errkit.WithCode(errkit.CodeInternal))
    }
    return u, nil
}

func getUser(c *gin.Context) {
    u, err := svc.Get(c, id)
    if err != nil {
        responsekit.GinError(c, err) // status from errkit/httperr, code from err.Code
        return
    }
    responsekit.GinOK(c, u)
}
```

### Handler that adds pagination

```go
type ListUsersResponse struct {
    Data       []User                  `json:"data"`
    Pagination paginationkit.OffsetResponse `json:"pagination"`
}

func listUsers(c *gin.Context) {
    req := paginationkit.NewOffsetPaginationRequest(parseOffset(c), parseLimit(c))
    users, total, err := svc.List(c, req)
    if err != nil {
        responsekit.GinError(c, err)
        return
    }
    page := ListUsersResponse{
        Data: users,
        Pagination: *paginationkit.NewOffsetPaginationResponse(
            req.Limit, &total, hasMore(req.Limit, total), req.Offset > 0,
        ),
    }
    responsekit.GinOK(c, page)
}
```

## Why three adapters?

The adapters exist so the *header / body contract* is owned by responsekit,
not by whichever router the service uses today. Migrating from Fiber to
Gin (or to `net/http`) is a one-line import swap inside `main.go` and a
rename of the helper calls — the wire shape and the test assertions stay
identical.

If you do not see your router here, look at the source — adding a new
adapter is a six-function file that delegates to `common.go`:

```go
func NewRouterOK(c NewRouterCtx, data any) error {
    return c.Status(200).JSON(successEnvelope(data))
}
```

## File Layout

| File | Purpose |
|------|---------|
| `common.go` | `Envelope`, `ErrorEnvelope`, `ErrorBody` wire shapes + `successEnvelope`, `errorEnvelope`, `statusCode` helpers |
| `gin.go` | `Gin*` helpers |
| `fiber.go` | `Fiber*` helpers |
| `nethttp.go` | `HTTP*` helpers + `writeJSON` (the only adapter that needs its own JSON encoder) |
| `responsekit_test.go` | All package tests (one matrix covers every adapter for parity) |
