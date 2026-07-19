package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/errkit/grpcerr"
	"github.com/hihand/go-platform/errkit/httperr"
	"github.com/hihand/go-platform/logkit"
	"github.com/hihand/go-platform/responsekit"
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

	// =============================================================================
	// 8. logkit: structured (JSON) logger
	// =============================================================================

	fmt.Println("\n--- logkit demo ---")

	// Build the root logger: INFO min-level, static service fields,
	// a mapper that lifts request.id out of context.
	logger := logkit.New(
		logkit.WithService("payment-api", "1.2.0"),
		logkit.WithDeployment("production"),
		logkit.WithContextMapper(func(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
			if v, ok := ctx.Value("request.id").(string); ok && v != "" {
				return append(attrs, logkit.String(logkit.AnyKey("request.id"), v))
			}
			return attrs
		}),
	)

	// 8a. Plain levels — call-site attrs merge into the record.
	logger.Debug("debug line", logkit.String(logkit.AnyKey("trace"), "abc"))
	logger.Info("user signed in",
		logkit.String(logkit.KeyEvent, "user.signed_in"),
		logkit.String(logkit.AnyKey("user.id"), "u-42"),
	)
	logger.Warn("retried",
		logkit.Int(logkit.AnyKey("attempts"), 3),
	)
	logger.Error("downstream failed",
		logkit.String(logkit.AnyKey("upstream"), "billing"),
	)

	// 8b. *Context variants — mapper is consulted, request.id lands
	// in the record. Debug is dropped because min-level is INFO.

	ctx := context.WithValue(context.Background(), "request.id", "req-7f3a")
	logger.DebugContext(ctx, "skipped (below min level)")
	logger.InfoContext(ctx, "payment created",
		logkit.String(logkit.KeyEvent, "payment.created"),
		logkit.String(logkit.AnyKey("payment.id"), "pay-001"),
		logkit.Int64(logkit.AnyKey("payment.amount"), 100),
	)

	// 8c. With — derive a scoped logger; child attrs override parent.
	scoped := logger.With(
		logkit.String(logkit.KeyEvent, "request.scoped"),
		logkit.String(logkit.AnyKey("request.id"), "scoped-req"),
	)
	scoped.InfoContext(ctx, "scoped log line")

	// 8d. Formatted variants. Debugf is skipped because the level
	// gate fires before Sprintf, so the format cost is avoided.
	logger.Debugf("dropped: %d %s", 1, "ignored")
	logger.Infof("user=%s age=%d", "alice", 30)
	logger.Errorf("payment failed: %s", "timeout")

	// 8e. A separate logger with caller capture on, scoped to DEBUG so
	// we can see what caller output looks like. The wrapper exists
	// because the immediate caller of logkit.Debug inside main is a
	// runtime frame; logging from a function on the call stack gives
	// a cleaner "main.go:line" output.
	debugLogger := logkit.New(
		logkit.WithService("payment-api", "1.2.0"),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithCaller(),
	)
	logFromHelper(debugLogger, "called from helper")

	// =============================================================================
	// 9. responsekit: HTTP response helpers (Gin adapter)
	// =============================================================================

	fmt.Println("\n--- responsekit demo ---")

	// Use a throwaway Gin engine + httptest recorder so we can show
	// each Gin* function's wire output without standing up a server.
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/users/:id", func(c *gin.Context) {
		switch c.Param("id") {
		case "42":
			responsekit.GinOK(c, map[string]string{"id": "42", "name": "alice"})
		case "99":
			responsekit.GinError(c, errkit.NotFound("user 99 not found"))
		case "500":
			responsekit.GinError(c, errors.New("upstream timed out"))
		default:
			responsekit.GinError(c, errkit.InvalidArgument("id must be a positive integer"))
		}
	})
	engine.POST("/users", func(c *gin.Context) {
		responsekit.GinCreated(c, map[string]string{"id": "u-new"})
	})
	engine.DELETE("/users/:id", func(c *gin.Context) {
		responsekit.GinNoContent(c)
	})

	for _, tc := range []struct {
		method, path string
	}{
		{"GET", "/users/42"},
		{"GET", "/users/99"},
		{"GET", "/users/500"},
		{"GET", "/users/-1"},
		{"POST", "/users"},
		{"DELETE", "/users/42"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		fmt.Printf("%-6s %-12s → %d %s\n", tc.method, tc.path, w.Code, strings.TrimSpace(w.Body.String()))
	}
}

func logFromHelper(l logkit.Logger, msg string) {
	l.Debug(msg)
}
