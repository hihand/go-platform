# HTTP Adapter (`errkit/httperr`)

Maps `errkit.Code` to HTTP status codes.

## Default Mapping

| errkit.Code | HTTP Status |
|-------------|-------------|
| `CodeInvalidArgument` | 400 Bad Request |
| `CodeNotFound` | 404 Not Found |
| `CodeAlreadyExists` | 409 Conflict |
| `CodeUnauthenticated` | 401 Unauthorized |
| `CodePermissionDenied` | 403 Forbidden |
| `CodeUnavailable` | 503 Service Unavailable |
| `CodeDeadlineExceeded` | 504 Gateway Timeout |
| `CodeCanceled` | 499 Client Closed Request |
| `CodeInternal` | 500 Internal Server Error |
| `CodeUnknown` | 500 Internal Server Error |

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

## Behavior

- `StatusCode(nil)` returns 500 (defaultStatus)
- Non-errkit errors return 500
- Unknown codes (including custom `Code` values) return 500
- The cause chain is walked via `errkit.FromError` — outer wraps do not affect the result
