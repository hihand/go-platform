# HTTP Adapter (`errkit/httperr`)

Maps `errkit.Code` to HTTP status codes.

## Default Mapping

Wire labels follow [RFC 9110 — HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html).
Status reasons below are the canonical IANA reason phrases.

### Transport / lifecycle

| errkit.Code | HTTP Status |
|-------------|-------------|
| `CodeCanceled` | 499 Client Closed Request (`httperr.StatusClientClosedRequest`) |
| `CodeDeadlineExceeded` | 504 Gateway Timeout |
| `CodeRequestTimeout` | 408 Request Timeout |

### Client errors (4xx)

| errkit.Code | HTTP Status |
|-------------|-------------|
| `CodeInvalidArgument` | 400 Bad Request |
| `CodeUnauthenticated` | 401 Unauthorized |
| `CodePermissionDenied` | 403 Forbidden |
| `CodeNotFound` | 404 Not Found |
| `CodeMethodNotAllowed` | 405 Method Not Allowed |
| `CodeNotAcceptable` | 406 Not Acceptable |
| `CodeConflict` | 409 Conflict |
| `CodeAlreadyExists` | 409 Conflict |
| `CodeGone` | 410 Gone |
| `CodeLengthRequired` | 411 Length Required |
| `CodePreconditionFailed` | 412 Precondition Failed |
| `CodePayloadTooLarge` | 413 Content Too Large |
| `CodeURITooLong` | 414 URI Too Long |
| `CodeUnsupportedMediaType` | 415 Unsupported Media Type |
| `CodeRangeNotSatisfiable` | 416 Range Not Satisfiable |
| `CodeExpectationFailed` | 417 Expectation Failed |
| `CodeMisdirectedRequest` | 421 Misdirected Request |
| `CodeUnprocessableEntity` | 422 Unprocessable Entity |
| `CodeLocked` | 423 Locked |
| `CodeFailedDependency` | 424 Failed Dependency |
| `CodeTooManyRequests` | 429 Too Many Requests |
| `CodeRequestHeaderFieldsTooLarge` | 431 Request Header Fields Too Large |
| `CodeUnavailableForLegalReasons` | 451 Unavailable For Legal Reasons |

### Server errors (5xx)

| errkit.Code | HTTP Status |
|-------------|-------------|
| `CodeInternal` | 500 Internal Server Error |
| `CodeNotImplemented` | 501 Not Implemented |
| `CodeBadGateway` | 502 Bad Gateway |
| `CodeUnavailable` | 503 Service Unavailable |
| `CodeDataLoss` | 507 Insufficient Storage |
| `CodeNetworkAuthenticationRequired` | 511 Network Authentication Required |

### Catch-all

| errkit.Code | HTTP Status |
|-------------|-------------|
| `CodeUnknown` | 500 Internal Server Error |

### Intentionally unmapped

`CodeDuplicate`, `CodePaymentRequired`, `CodeUpgradeRequired`, and every
custom code the library does not know about (e.g. `Code("DOMAIN_X")`) fall
back to **500 Internal Server Error**. Override via `NewMapper`:

```go
m := httperr.NewMapper(map[errkit.Code]int{
    errkit.CodeDuplicate:        http.StatusConflict,
    errkit.CodePaymentRequired:  http.StatusPaymentRequired,
    errkit.CodeUpgradeRequired:  http.StatusUpgradeRequired,
    errors.Code("DOMAIN_X"):     http.StatusFailedDependency, // example
})
```

## Usage

### Package-level helper

```go
import "github.com/hihand/go-platform/errkit/httperr"

err := errkit.NotFound("user 42")
status := httperr.StatusCode(err)
// status == 404
```

### Custom mapper

Use `NewMapper` when you need to override specific mappings. The mapper is immutable after construction and safe for concurrent use.

```go
import "github.com/hihand/go-platform/errkit/httperr"

mapper := httperr.NewMapper(map[errkit.Code]int{
    errkit.CodeNotFound: 200,  // override: return 200 instead of 404
})

status := mapper.StatusCode(errkit.NotFound("user 42"))
// status == 200
```

## In an HTTP Handler

```go
func getUser(w http.ResponseWriter, r *http.Request) {
    user, err := userService.Get(r.Context(), userID)
    if err != nil {
        status := httperr.StatusCode(err)
        writeJSON(w, status, map[string]any{
            "error": map[string]any{
                "code":    errkit.CodeOf(err),
                "message": errkit.MessageOf(err),
                "meta":    errkit.MetadataOf(err),
            },
        })
        return
    }
    writeJSON(w, http.StatusOK, user)
}
```

## Behaviour

- `StatusCode(nil)` returns 500 (`defaultStatus`)
- Non-errkit errors return 500
- Unknown codes (including custom `Code` values, and built-ins
  `CodeDuplicate`, `CodePaymentRequired`, `CodeUpgradeRequired`) return 500
- The cause chain is walked via `errkit.FromError` — outer wraps do not affect the result

## Sibling packages

- [logkit](logkit.md) — structured JSON logger
- [responsekit](responsekit.md) — unified HTTP response envelope (Gin / Fiber / net/http)
- [paginationkit](paginationkit.md) — offset / cursor pagination models
