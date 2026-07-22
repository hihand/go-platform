# gRPC Adapter (`errkit/grpcerr`)

Maps `errkit.Code` to gRPC codes and builds `google.golang.org/grpc/status.Status` values.

## Default Mapping

Wire labels follow the [gRPC status codes specification](https://grpc.io/docs/grpc-framework/status-codes/).

### Transport / lifecycle

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeCanceled` | `codes.Canceled` |
| `CodeDeadlineExceeded` | `codes.DeadlineExceeded` |
| `CodeRequestTimeout` | `codes.DeadlineExceeded` (no dedicated gRPC code) |

### Client errors → `InvalidArgument`

These all surface as `codes.InvalidArgument` (gRPC has no finer-grained
"structural" code):

| errkit.Code |
|-------------|
| `CodeInvalidArgument` |
| `CodeUnprocessableEntity` |
| `CodeMethodNotAllowed` |
| `CodeURITooLong` |
| `CodeExpectationFailed` |
| `CodeMisdirectedRequest` |
| `CodeNotAcceptable` |
| `CodeLengthRequired` |
| `CodeUnsupportedMediaType` |

### Client errors → `Unauthenticated` / `PermissionDenied`

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeUnauthenticated` | `codes.Unauthenticated` |
| `CodePermissionDenied` | `codes.PermissionDenied` |

### Client errors → `FailedPrecondition`

When the operation *could* succeed if the resource state were different:

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeLocked` | `codes.FailedPrecondition` |
| `CodeFailedDependency` | `codes.FailedPrecondition` |
| `CodeUnavailableForLegalReasons` | `codes.FailedPrecondition` |
| `CodePreconditionFailed` | `codes.FailedPrecondition` |

### Client errors → resource state / conflict

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeNotFound` | `codes.NotFound` |
| `CodeGone` | `codes.NotFound` |
| `CodeAlreadyExists` | `codes.AlreadyExists` |
| `CodeConflict` | `codes.Aborted` |

### Client errors → `OutOfRange`

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeRangeNotSatisfiable` | `codes.OutOfRange` |

### Client errors → `ResourceExhausted`

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeTooManyRequests` | `codes.ResourceExhausted` |
| `CodePayloadTooLarge` | `codes.ResourceExhausted` |
| `CodeRequestHeaderFieldsTooLarge` | `codes.ResourceExhausted` |

### Server errors

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeInternal` | `codes.Internal` |
| `CodeNotImplemented` | `codes.Unimplemented` |
| `CodeBadGateway` | `codes.Unavailable` |
| `CodeUnavailable` | `codes.Unavailable` |
| `CodeDataLoss` | `codes.DataLoss` |
| `CodeNetworkAuthenticationRequired` | `codes.Unauthenticated` |

### Catch-all

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeUnknown` | `codes.Unknown` |

### Intentionally unmapped

`CodeDuplicate`, `CodePaymentRequired`, `CodeUpgradeRequired`, and every
custom code the library does not know about fall back to `codes.Unknown`.
Override via `NewMapper`:

```go
m := grpcerr.NewMapper(map[errkit.Code]codes.Code{
    errkit.CodeDuplicate:        codes.AlreadyExists,
    errkit.CodePaymentRequired:  codes.FailedPrecondition,
    errkit.Code("DOMAIN_X"):     codes.ResourceExhausted, // example
})
```

## Usage

### Package-level helper

```go
import (
    "github.com/hihand/go-platform/errkit"
    "github.com/hihand/go-platform/errkit/grpcerr"
)

err := errkit.Internal("database unavailable")
grpcStatus := grpcerr.ToGRPCStatus(err)
// grpcStatus.Code() == codes.Internal
// grpcStatus.Message() == "database unavailable"
```

### Return as gRPC error

```go
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    user, err := s.svc.Get(ctx, req.GetId())
    if err != nil {
        return nil, grpcerr.ToGRPCError(err)
    }
    return user, nil
}
```

`ToGRPCError(nil)` returns `nil` — safe to use directly in handlers without nil checks.

### Custom mapper

```go
mapper := grpcerr.NewMapper(map[errkit.Code]codes.Code{
    errkit.CodeNotFound: codes.FailedPrecondition, // override
})

status := mapper.ToGRPCStatus(errkit.NotFound("user 42"))
// status.Code() == codes.FailedPrecondition
```

## Two Return Forms

| Function | Returns | Use case |
|----------|---------|----------|
| `ToGRPCStatus` | `*status.Status` | Inspect code/message, build trailers, compose |
| `ToGRPCError` | `error` | Return directly from a gRPC handler |

```go
st := grpcerr.ToGRPCStatus(err)
// st is a *status.Status — you can add details, build trailers, etc.

grpcErr := grpcerr.ToGRPCError(err)
// grpcErr is an error suitable for: return nil, grpcErr
```

## Checking gRPC Code from the Client Side

After a gRPC call returns an error, use `google.golang.org/grpc/status` to check the code:

```go
import "google.golang.org/grpc/status"

st, ok := status.FromError(grpcErr)
if ok {
    fmt.Println(st.Code())   // e.g., codes.Internal
    fmt.Println(st.Message()) // e.g., "database unavailable"
}
```

Or use `errors.Is` / `errors.As` from `google.golang.org/grpc/codes`:

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

if status.Code(grpcErr) == codes.Internal {
    // handle internal error
}
```

## Behaviour

- `ToGRPCStatus(nil)` returns a status with `codes.Unknown` and empty message
- `ToGRPCError(nil)` returns `nil`
- The `Status.Message` is set to `errkit.MessageOf(err)` — the errkit message, not the underlying cause
- The cause chain is walked via `errkit.FromError`

## Sibling packages

- [logkit](logkit.md) — structured JSON logger
- [responsekit](responsekit.md) — unified HTTP response envelope (Gin / Fiber / net/http)
- [paginationkit](paginationkit.md) — offset / cursor pagination models
