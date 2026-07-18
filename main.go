package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/errkit/grpcerr"
	"github.com/hihand/go-platform/errkit/httperr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {

	// =============================================================================
	// 1. Core: New + Options
	// =============================================================================

	// Full construction with all options
	err := errkit.New(
		errkit.WithCode(errkit.CodeInvalidArgument),
		errkit.WithMessage("field 'email' is required"),
		errkit.WithMetadata(map[string]any{"field": "email", "value": ""}),
	)
	fmt.Println(err)
	// => INVALID_ARGUMENT: field 'email' is required

	// =============================================================================
	// 2. Sugar constructors (shortcut for common codes)
	// =============================================================================

	err = errkit.NotFound("user 42 not found")
	fmt.Println(err)
	// => NOT_FOUND: user 42 not found

	err = errkit.Internal("database connection refused")
	fmt.Println(err)
	// => INTERNAL: database connection refused

	// =============================================================================
	// 3. Wrap + errors.Is / errors.As
	// =============================================================================

	cause := errors.New("connection refused")
	err = errkit.Wrap(cause,
		errkit.WithCode(errkit.CodeUnavailable),
		errkit.WithMessage("upstream is down"),
	)
	fmt.Println(err)
	// => UNAVAILABLE: upstream is down: connection refused

	fmt.Println(errors.Is(err, cause))                      // true — cause chain preserved
	fmt.Println(errkit.IsCode(err, errkit.CodeUnavailable)) // true

	// =============================================================================
	// 4. Predicates: CodeOf, MessageOf, MetadataOf, FromError
	// =============================================================================

	fmt.Println(errkit.CodeOf(err))     // UNAVAILABLE
	fmt.Println(errkit.MessageOf(err))  // upstream is down
	fmt.Println(errkit.MetadataOf(err)) // map[] (no metadata on this error)

	e, ok := errkit.FromError(err)
	fmt.Printf("found=%v, code=%s\n", ok, e.Code())

	// =============================================================================
	// 5. httperr adapter
	// =============================================================================

	errHTTP := errkit.NotFound("user 42")
	fmt.Printf("HTTP status: %d (%s)\n", httperr.StatusCode(errHTTP), http.StatusText(httperr.StatusCode(errHTTP)))
	// => HTTP status: 404 (Not Found)

	// Custom mapper with overrides
	mapper := httperr.NewMapper(map[errkit.Code]int{
		errkit.CodeNotFound: 200, // override: treat NotFound as "resource gone but ok"
	})
	fmt.Printf("Custom HTTP status: %d\n", mapper.StatusCode(errHTTP))
	// => HTTP status: 200

	// =============================================================================
	// 6. grpcerr adapter
	// =============================================================================

	errGRPC := errkit.Internal("database connection refused")
	grpcStatus := grpcerr.ToGRPCStatus(errGRPC)
	fmt.Printf("gRPC code: %v, message: %s\n", grpcStatus.Code(), grpcStatus.Message())
	// => gRPC code: Internal, message: database connection refused

	// Convert to gRPC error to return from handler
	grpcErr := grpcerr.ToGRPCError(errGRPC)
	fmt.Println(grpcErr)
	// => rpc error: code = Internal desc = database connection refused

	// Check gRPC code using errors.Is
	fmt.Println(status.Code(grpcErr) == codes.Internal) // true

	// =============================================================================
	// 7. graphqlerr adapter
	// =============================================================================

	// (see errkit/graphqlerr/graphqlerr.go)
	// gqlErr := graphqlerr.ToGraphQLError(err)

}
