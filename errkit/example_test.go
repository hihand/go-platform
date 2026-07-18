package errkit_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/grpc/codes"

	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/errkit/grpcerr"
	"github.com/hihand/go-platform/errkit/httperr"
)

// ExampleNew demonstrates the canonical options-based construction.
func ExampleNew() {
	err := errkit.New(
		errkit.WithCode(errkit.CodeInvalidArgument),
		errkit.WithMessage("id is required"),
		errkit.WithMetadata(map[string]any{
			"field": "id",
		}),
	)
	fmt.Println(err.Code())
	fmt.Println(err.Message())
	fmt.Println(errkit.MetadataOf(err)["field"])
	// Output:
	// INVALID_ARGUMENT
	// id is required
	// id
}

// ExampleNotFound shows the NotFound sugar constructor.
func ExampleNotFound() {
	fmt.Println(errkit.NotFound("user 42"))
	// Output:
	// NOT_FOUND: user 42
}

// ExampleInvalidArgument shows the InvalidArgument sugar constructor.
func ExampleInvalidArgument() {
	fmt.Println(errkit.InvalidArgument("id is required"))
	// Output:
	// INVALID_ARGUMENT: id is required
}

// ExampleInternal shows the Internal sugar constructor.
func ExampleInternal() {
	fmt.Println(errkit.Internal("database unavailable"))
	// Output:
	// INTERNAL: database unavailable
}

// ExampleWrap shows how to attach errkit attributes to an existing error
// while keeping errors.Is/As compatibility with the original cause.
func ExampleWrap() {
	base := io.EOF
	err := errkit.Wrap(base,
		errkit.WithCode(errkit.CodeUnavailable),
		errkit.WithMessage("upstream is down"),
	)
	fmt.Println(errkit.CodeOf(err) == errkit.CodeUnavailable)
	fmt.Println(errors.Is(err, base))
	fmt.Println(err.Error())
	// Output:
	// true
	// true
	// UNAVAILABLE: upstream is down: EOF
}

// Example_httperr_StatusCode shows how to translate an errkit error into an HTTP status.
func Example_httperr_StatusCode() {
	err := errkit.NotFound("user 42")
	fmt.Println(httperr.StatusCode(err) == http.StatusNotFound)
	// Output:
	// true
}

// Example_grpcerr_ToGRPCStatus shows how to translate an errkit error into a gRPC status.
func Example_grpcerr_ToGRPCStatus() {
	st := grpcerr.ToGRPCStatus(errkit.InvalidArgument("id is required"))
	fmt.Println(st.Code() == codes.InvalidArgument)
	fmt.Println(st.Message())
	// Output:
	// true
	// id is required
}
