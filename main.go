package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hihand/go-platform/configkit"
	"github.com/hihand/go-platform/contextkit"
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

	// =============================================================================
	// 10. configkit: YAML file + env-var overrides, with validation
	// =============================================================================

	fmt.Println("\n--- configkit demo ---")

	// Drop a small YAML file into a temp dir so the demo is
	// self-contained — no fixture file committed to the repo.
	demoDir, mkdirErr := os.MkdirTemp("", "configkit-demo-")
	if mkdirErr != nil {
		fmt.Println("mkdir:", mkdirErr)
		return
	}
	defer os.RemoveAll(demoDir)

	demoYAML := filepath.Join(demoDir, "config.yaml")
	if writeErr := os.WriteFile(demoYAML, []byte(`
server:
  host: 0.0.0.0
  port: 8080
database:
  url: postgres://localhost/app
  max_conns: 5
log:
  level: info
`), 0o600); writeErr != nil {
		fmt.Println("write yaml:", writeErr)
		return
	}

	// 10a. Full wiring — file + env + prefix + defaults + validator.
	//
	//	APP_SERVER_HOST=api.example.com \
	//	APP_SERVER_PORT=9090 \
	//	go run main.go
	//
	// re-runs with overrides. Without the env vars the file
	// values stand.
	loader := configkit.New(
		configkit.WithConfigFile(demoYAML),
		configkit.WithEnv(),
		configkit.WithEnvPrefix("APP"),
		configkit.WithDefaults(map[string]any{
			"server.host":   "fallback.example.com",
			"server.port":   4000,
			"database.url":  "postgres://localhost/app",
			"database.conn": 5,
			"log.level":     "warn",
		}),
		configkit.WithValidator(func(c any) error {
			// Catch missing values that the file + env left blank.
			// Real apps wire a struct validator here
			// (go-playground/validator, ozzo-validation, ...).
			if c.(*DemoAppConfig).Server.Port == 0 {
				return errors.New("server.port is required")
			}
			return nil
		}),
	)

	var cfg DemoAppConfig
	if err := loader.Load(&cfg); err != nil {
		fmt.Println("load:", err)
		return
	}

	fmt.Printf("server  = %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("database = %s (max_conns=%d)\n", cfg.Database.URL, cfg.Database.MaxConns)
	fmt.Printf("log     = %s\n", cfg.Log.Level)

	// 10b. Precedence matrix — env > file > default. The Loader
	// itself doesn't expose which source supplied a value, so we
	// re-run it twice: once without env to capture the file/default
	// baseline, and once with env set to capture the override.
	fmt.Println("\n--- configkit precedence (env > file > default) ---")

	baseline := func() DemoAppConfig {
		var b DemoAppConfig
		_ = configkit.New(
			configkit.WithConfigFile(demoYAML),
			configkit.WithDefaults(map[string]any{
				"server.host":  "fallback.example.com",
				"server.port":  4000,
				"database.url": "postgres://localhost/app",
				"log.level":    "warn",
			}),
		).Load(&b)
		return b
	}()

	// Set one env var so the override path is exercised; remember
	// to restore it after the demo so a re-run is deterministic.
	const overrideKey = "APP_SERVER_PORT"
	const overrideVal = "9090"
	prevOverride, hadOverride := os.LookupEnv(overrideKey)
	_ = os.Setenv(overrideKey, overrideVal)
	var overridden DemoAppConfig
	_ = configkit.New(
		configkit.WithConfigFile(demoYAML),
		configkit.WithEnv(),
		configkit.WithEnvPrefix("APP"),
		configkit.WithDefaults(map[string]any{
			"server.host":  "fallback.example.com",
			"server.port":  4000,
			"database.url": "postgres://localhost/app",
			"log.level":    "warn",
		}),
	).Load(&overridden)
	if hadOverride {
		_ = os.Setenv(overrideKey, prevOverride)
	} else {
		_ = os.Unsetenv(overrideKey)
	}

	sourceFor := func(key string, baselineVal, overrideVal string) string {
		if baselineVal != overrideVal {
			return "env"
		}
		if baselineVal == defaultValueFor(key) {
			return "default"
		}
		return "file"
	}
	for _, key := range []string{"server.host", "server.port", "log.level"} {
		base := cfgValue(baseline, key)
		over := cfgValue(overridden, key)
		fmt.Printf("%-12s ← %-7s (file=%s, default=%s, env=%s)\n",
			key, sourceFor(key, base, over),
			base, defaultValueFor(key), over,
		)
	}

	// 10c. Missing config file is silently ignored — boot from
	// env + defaults alone. Useful for local development where the
	// app may not have a config file yet.
	fmt.Println("\n--- configkit: missing file is ignored ---")
	missingPath := filepath.Join(demoDir, "absent.yaml")
	var bare DemoAppConfig
	loadErr := configkit.New(
		configkit.WithConfigFile(missingPath),
		configkit.WithDefault("server.host", "localhost"),
		configkit.WithDefault("server.port", 3000),
	).Load(&bare)
	fmt.Println("err    =", loadErr)
	fmt.Println("server =", bare.Server.Host+":"+itoa(bare.Server.Port))

	// 10d. Validator surfaces errors verbatim. The default
	// struct port is 0 here, so the validator fires.
	fmt.Println("\n--- configkit: validator failure ---")
	validateErr := configkit.New(
		configkit.WithValidator(func(c any) error {
			if c.(*DemoAppConfig).Server.Port == 0 {
				return errors.New("server.port is required")
			}
			return nil
		}),
	).Load(&DemoAppConfig{})
	fmt.Println("err    =", validateErr)

	// =============================================================================
	// 11. contextkit: request-scoped metadata via context.Context
	// =============================================================================

	fmt.Println("\n--- contextkit demo ---")

	// 11a. WithRequest + GetRequest — round-trip a request-scoped
	// struct. The returned value is a copy, so callers can mutate
	// it locally without affecting other layers.
	ctx = contextkit.WithRequest(context.Background(), contextkit.Request{
		RequestID: "req-7f3a",
		TraceID:   "trace-001",
		SpanID:    "span-002",
	})
	req := contextkit.GetRequest(ctx)
	fmt.Printf("request = %+v\n", req)

	// Overwrite on a derived context leaves the parent untouched.
	overwrittenCtx := contextkit.WithRequest(ctx, contextkit.Request{
		RequestID: "req-NEW",
	})
	fmt.Println("parent  =", contextkit.GetRequest(ctx).RequestID)
	fmt.Println("child   =", contextkit.GetRequest(overwrittenCtx).RequestID)

	// Missing values are reported as the zero struct, never as a panic.
	fmt.Println("missing =", contextkit.GetRequest(context.Background()) == (contextkit.Request{}))

	// 11b. WithIdentity + Identity — a generic, type-safe carrier
	// for whatever auth-shaped payload the platform propagates
	// downstream. The package has no opinion on the struct shape.
	ctx = contextkit.WithIdentity(ctx, demoClaims{
		UserID:     "u-42",
		MerchantID: "m-7",
		Roles:      []string{"admin", "billing"},
	})
	c, ok := contextkit.Identity[demoClaims](ctx)
	fmt.Printf("identity ok=%v user=%s merchant=%s roles=%v\n",
		ok, c.UserID, c.MerchantID, c.Roles)

	// Asking with the wrong T returns (zero, false) — no panic.
	_, wrong := contextkit.Identity[demoOtherClaims](ctx)
	fmt.Println("wrong type ok =", wrong)

	// Asking on a context that never had an identity returns the
	// same (zero, false).
	_, missing := contextkit.Identity[demoClaims](context.Background())
	fmt.Println("missing identity ok =", missing)

	// 11c. Pipeline — a transport edge stamps Request + Identity,
	// a downstream handler reads them back. The handler is a
	// regular function: contextkit is just a convention. We
	// re-derive a context that keeps the freshly stamped
	// Identity and the freshly stamped Request so the demo
	// shows the full pipeline in one call.
	downstreamCtx := contextkit.WithRequest(ctx, contextkit.Request{
		RequestID: "req-NEW",
		TraceID:   "trace-099",
		SpanID:    "span-100",
	})
	handlePayment(downstreamCtx)
}

func logFromHelper(l logkit.Logger, msg string) {
	l.Debug(msg)
}

// ---------- configkit demo helpers ----------------------------------------

// DemoAppConfig is the application-owned configuration struct used
// by the configkit demo. It belongs to main.go, not to configkit —
// the package has no knowledge of it.
type DemoAppConfig struct {
	Server   DemoServer   `mapstructure:"server"`
	Database DemoDatabase `mapstructure:"database"`
	Log      DemoLog      `mapstructure:"log"`
}

type DemoServer struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DemoDatabase struct {
	URL      string `mapstructure:"url"`
	MaxConns int    `mapstructure:"max_conns"`
}

type DemoLog struct {
	Level string `mapstructure:"level"`
}

// cfgValue returns the string form of a nested config field by
// dotted path. Used by the precedence demo to print the winning
// value for a key without dragging in reflection.
func cfgValue(c DemoAppConfig, key string) string {
	switch key {
	case "server.host":
		return c.Server.Host
	case "server.port":
		return itoa(c.Server.Port)
	case "database.url":
		return c.Database.URL
	case "database.max_conns":
		return itoa(c.Database.MaxConns)
	case "log.level":
		return c.Log.Level
	}
	return "<unknown>"
}

// defaultValueFor mirrors the WithDefaults map in the demo so the
// precedence table can label each value's source (file vs. default
// vs. env). Kept in sync with the option call above.
func defaultValueFor(key string) string {
	switch key {
	case "server.host":
		return "fallback.example.com"
	case "server.port":
		return "4000"
	case "log.level":
		return "warn"
	}
	return "<unknown>"
}

// itoa keeps the demo self-contained — strconv.Itoa would also do,
// but a tiny wrapper makes the helper block above easier to read.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ---------- contextkit demo helpers --------------------------------------

// demoClaims is a stand-in for whatever auth-shaped struct a
// production codebase would propagate via contextkit.Identity. The
// package has no opinion on the shape; this one mirrors the
// Claims example in the package documentation.
type demoClaims struct {
	UserID     string
	MerchantID string
	Roles      []string
}

// demoOtherClaims shares a single field name with demoClaims but
// is otherwise unrelated. It exists only so the demo can show that
// reading with the wrong T surfaces as (zero, false) — no panic.
type demoOtherClaims struct {
	UserID string
}

// handlePayment is the "downstream layer" the contextkit demo
// hands a context to. In a real codebase it would be a gRPC handler,
// HTTP middleware, repository method, or queue worker. The body
// here only proves that contextkit values survive the call.
func handlePayment(ctx context.Context) {
	req := contextkit.GetRequest(ctx)
	c, ok := contextkit.Identity[demoClaims](ctx)
	if !ok {
		fmt.Println("handlePayment: no identity, skipping")
		return
	}
	fmt.Printf("handlePayment: request_id=%s user=%s merchant=%s\n",
		req.RequestID, c.UserID, c.MerchantID)
}
