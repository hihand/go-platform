# paginationkit

Lightweight, transport-agnostic pagination models for the go-platform ecosystem. Stdlib-only.

## At a glance

```go
// Build a request from raw transport input. Bad limit/offset is normalised here.
req := paginationkit.NewOffsetPaginationRequest(-5, 9999)
// req.Limit  == 100  (clamped to MaxPaginationLimit)
// req.Offset == 0    (clamped to 0)

// Build the matching response. Optional fields are pointers, so "absent"
// is distinguishable from "zero".
total := int64(137)
resp := paginationkit.NewOffsetPaginationResponse(20, &total, true, false)
// resp.Total       = &137
// resp.HasNext     = true
// resp.HasPrevious = false
```

## Design contract

- **stdlib only** — no HTTP, gRPC, GraphQL, ORM, or SQL dependency. The
  package owns pagination *models*, not query parsing or DB slicing.
- **Normalise at construction.** `limit <= 0` becomes `DefaultPaginationLimit`
  (20); `limit > MaxPaginationLimit` (100) is clamped; negative offsets are
  clamped to 0. Downstream code can trust what it reads.
- **Optional fields are pointers.** `Total *int64`, `NextCursor *string`,
  `PrevCursor *string` — so `"absent"` is distinguishable from `"zero"`.
- **Explicit constructor names.** `NewOffsetPaginationRequest`,
  `NewCursorPaginationResponse`, … never any positional surprises.
- **No global state.** No `init()`, no package-level mutability.
- **No reflection.** All fields are exported, stable, JSON-tagged.

## When to use offset, when to use cursor

| Use `OffsetRequest` / `OffsetResponse` when | Use `CursorRequest` / `CursorResponse` when |
|---|---|
| The dataset is small or bounded. | The dataset is large or unbounded (logs, events, feeds). |
| The UI needs page numbers / page counts. | Rows can be inserted or deleted while paging (offset would skip or repeat). |
| The caller is happy with deeper pages being slower (`OFFSET N` scans). | Latency must stay constant regardless of paging depth. |

Cursor pagination is also a better fit for clients that page forever — the
producer can return an opaque token that the next request can hand back
without leaking the sort key.

## Constants

```go
const DefaultPaginationLimit int = 20
const MaxPaginationLimit     int = 100
```

Both live in `common.go` and are the only knobs you set once globally. The
constructors apply them automatically; if you need a different ceiling, build
your own wrapper that calls `NewOffsetPaginationRequest` / `NewCursorPaginationRequest`
after clamping yourself.

## Request models

```go
type BaseRequest struct {
    Limit int `json:"limit"` // [1, MaxPaginationLimit], normalised at construction
}

type OffsetRequest struct {
    BaseRequest
    Offset int `json:"offset"` // clamped to >= 0 at construction
}

type CursorRequest struct {
    BaseRequest
    After  string `json:"after,omitempty"`  // opaque to this package
    Before string `json:"before,omitempty"` // opaque to this package
}
```

`Offset` and `Cursor` requests **embed** `BaseRequest` rather than duplicate
the `Limit` field, so the limit clamping lives in exactly one place
(`common.go:normaliseLimit`). Add a third style (say, page-number
pagination) by embedding `BaseRequest` again.

## Response models

```go
type BaseResponse struct {
    Limit       int  `json:"limit"`
    HasNext     bool `json:"has_next"`
    HasPrevious bool `json:"has_previous"`
}

type OffsetResponse struct {
    BaseResponse
    Total *int64 `json:"total,omitempty"`
}

type CursorResponse struct {
    BaseResponse
    NextCursor *string `json:"next_cursor,omitempty"`
    PrevCursor *string `json:"prev_cursor,omitempty"`
}
```

`Total` on `OffsetResponse`:

| `Total` | Meaning on the wire |
|---------|---------------------|
| `nil`   | omitted (`json:"total,omitempty"`) — service did not run `COUNT(*)` |
| `&0`    | emitted as `0` — service confirmed there are no items |

Same pattern for `NextCursor` / `PrevCursor`: pointers let you emit
`"next_cursor":""` if you genuinely have an empty cursor, while distinguishing
that from a `nil` cursor meaning *"no next page"*.

## Constructors

### Offset

```go
// NewOffsetPaginationRequest(offset, limit int) OffsetRequest
req := paginationkit.NewOffsetPaginationRequest(50, 10) // second page of 10
req := paginationkit.NewOffsetPaginationRequest(-5, 9999)
// req.Offset == 0    (negative clamped)
// req.Limit  == 100  (over MaxPaginationLimit)

// NewOffsetPaginationResponse(limit int, total *int64, hasNext, hasPrevious bool) OffsetResponse
total := int64(137)
resp := paginationkit.NewOffsetPaginationResponse(20, &total, true, false)
resp := paginationkit.NewOffsetPaginationResponse(20, nil, false, false) // Total absent
```

### Cursor

```go
// NewCursorPaginationRequest(after, before string, limit int) CursorRequest
req := paginationkit.NewCursorPaginationRequest("",      "",      25) // first page, default limit
req := paginationkit.NewCursorPaginationRequest("eyJpZCI6MTB9", "",      50) // forward from cursor
req := paginationkit.NewCursorPaginationRequest("",      "eyJpZCI6NTB9", 25) // backward from cursor

// NewCursorPaginationResponse(limit int, nextCursor, prevCursor *string, hasNext, hasPrevious bool) CursorResponse
next := "eyJpZCI6MjB9"
prev := "eyJpZCI6MTAifQ"
resp := paginationkit.NewCursorPaginationResponse(10, &next, &prev, true, true)
```

The cursor strings are **opaque to this package**. paginationkit never
parses, validates, or encodes them — it stores them verbatim. Whatever
the producer decides (base64-encoded offset, `{id, ts}` JSON, signed
token, …), the constructor will accept it.

### Sugar

The most common cases have one-liners so handlers stay compact:

```go
paginationkit.FirstPage(15)              // OffsetRequest with offset=0, normalised limit
paginationkit.EmptyOffsetResponse(20)    // zero-page offset response
paginationkit.EmptyCursorResponse(20)    // zero-page cursor response
```

## Cursor encoding — by example

A producer decides what a cursor means. A typical choice:

```go
import "encoding/base64"
import "encoding/json"

type cursor struct {
    ID int64 `json:"id"`
}

// forward
next := base64.StdEncoding.EncodeToString(mustJSON(cursor{ID: last.ID}))
resp := paginationkit.NewCursorPaginationResponse(
    limit,
    &next,   // NextCursor
    nil,     // PrevCursor
    true,    // hasNext
    false,   // hasPrevious
)

// backward — same response, different cursor field
prev := base64.StdEncoding.EncodeToString(mustJSON(cursor{ID: first.ID}))
resp := paginationkit.NewCursorPaginationResponse(
    limit,
    nil,
    &prev,
    false,
    true,
)
```

paginationkit does not ship an encoder — pick the encoding that fits your
data and keep the constructor for the cursor field. The package will not
try to second-guess you.

## Errkit integration

Out of the box, paginationkit does **not** raise `errkit.Error`. Constructors
are total: invalid input is clamped, not rejected. Two natural pairings
when using the pagination models inside an `errkit`-aware service:

```go
func (h *handler) ListUsers(c *gin.Context) {
    req, err := parseFromQuery(c) // your HTTP query parser
    if err != nil {
        responsekit.GinError(c, errkit.InvalidArgument(err.Error()))
        return
    }

    users, err := h.svc.List(c.Request.Context(), req)
    if err != nil {
        responsekit.GinError(c, err)
        return
    }
    responsekit.GinOK(c, users) // wrap with pagination shape as you prefer
}
```

```go
type Page struct {
    Data  any                          `json:"data"`
    Pagination paginationkit.OffsetResponse `json:"pagination"`
}

func (h *handler) ListUsers(c *gin.Context) {
    var req paginationkit.OffsetRequest
    // parse into req ...
    items, total, err := h.svc.List(c.Request.Context(), req)
    // ...
    page := Page{Data: items, Pagination: *paginationkit.NewOffsetPaginationResponse(
        req.Limit, &total, hasMore, req.Offset > 0,
    )}
    responsekit.GinOK(c, page)
}
```

The `*int64` / `*string` optional fields keep "absent" honest on the wire,
which matters when client SDKs use `total != nil` to decide whether to
render a "Page N of M" widget.

## File Layout

| File | Purpose |
|------|---------|
| `spec.go` | Package doc comment; design contract + file layout |
| `common.go` | `DefaultPaginationLimit`, `MaxPaginationLimit`, `normaliseLimit`, `normaliseOffset` |
| `request.go` | `BaseRequest`, `OffsetRequest`, `CursorRequest` |
| `response.go` | `BaseResponse`, `OffsetResponse`, `CursorResponse` |
| `offset.go` | `NewOffsetPaginationRequest`, `NewOffsetPaginationResponse` |
| `cursor.go` | `NewCursorPaginationRequest`, `NewCursorPaginationResponse` |
| `new.go` | Sugar: `FirstPage`, `EmptyOffsetResponse`, `EmptyCursorResponse` |
| `paginationkit_test.go` | All package tests |
| `example_test.go` | Runnable godoc examples |
