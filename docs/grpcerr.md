# gRPC Adapter (`errkit/grpcerr`)

Maps `errkit.Code` to gRPC codes and builds `google.golang.org/grpc/status.Status` values.

## Default Mapping

| errkit.Code | gRPC Code |
|-------------|-----------|
| `CodeInvalidArgument` | `codes.InvalidArgument` |
| `CodeNotFound` | `codes.NotFound` |
| `CodeAlreadyExists` | `codes.AlreadyExists` |
| `CodeUnauthenticated` | `codes.Unauthenticated` |
| `CodePermissionDenied` | `codes.PermissionDenied` |
| `CodeUnavailable` | `codes.Unavailable` |
| `CodeDeadlineExceeded` | `codes.DeadlineExceeded` |
| `CodeCanceled` | `codes.Canceled` |
| `CodeInternal` | `codes.Internal` |
| `CodeUnknown` | `codes.Unknown` |

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
|----------|---------|---------|
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
    fmt.Println(st.Code()) // e.g., codes.Internal
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

## Behavior

- `ToGRPCStatus(nil)` returns a status with `codes.Unknown` and empty message
- `ToGRPCError(nil)` returns `nil`
- The `Status.Message` is set to `errkit.MessageOf(err)` — the errkit message, not the underlying cause
- The cause chain is walked via `errkit.FromError`
