package logkit_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hihand/go-platform/logkit"
)

// ExampleNew shows the canonical logger construction with static
// resource fields. Keys come from the typed enum in spec.go;
// application-defined correlation/business fields use AnyKey.
func ExampleNew() {
	var buf captureBuffer
	logger := logkit.New(
		logkit.WithOutput(&buf),
		logkit.WithService("payment-api", "1.2.0"),
		logkit.WithDeployment("production"),
	)
	logger.InfoContext(
		context.WithValue(context.Background(), reqIDKey, "req-123"),
		"payment created",
		logkit.String(logkit.KeyEvent, "payment.created"),
		logkit.String(logkit.AnyKey("payment.id"), "pay-001"),
		logkit.Int64(logkit.AnyKey("payment.amount"), 100),
	)
	out := buf.String()
	fmt.Println(strContains(out, `"event":"payment.created"`,
		`"service.name":"payment-api"`,
		`"service.version":"1.2.0"`,
		`"deployment.environment":"production"`,
		`"payment.id":"pay-001"`,
		`"payment.amount":100`,
	))
	// Output:
	// true
}

// ExampleWithContextMapper demonstrates the application's own mapper
// owning all correlation extraction. logkit itself never reads the
// context — the mapper receives it once per *Context call.
func ExampleWithContextMapper() {
	var buf captureBuffer
	logger := logkit.New(
		logkit.WithOutput(&buf),
		logkit.WithContextMapper(defaultReqIDMapper),
	)
	ctx := withReqID(context.Background(), "req-abc")
	logger.InfoContext(ctx, "in span")
	fmt.Println(strContains(buf.String(), `"request.id":"req-abc"`))
	// Output:
	// true
}

// ExampleLogger_with demonstrates With: deriving a scoped logger from a
// root.
func ExampleLogger_with() {
	var buf captureBuffer
	root := logkit.New(logkit.WithOutput(&buf))
	scoped := root.With(logkit.String(logkit.AnyKey("request.id"), "req-with"))
	scoped.Info("scoped event")
	fmt.Println(strContains(buf.String(), `"request.id":"req-with"`))
	// Output:
	// true
}

// Example_withCaller covers WithCaller enabling caller capture.
func Example_withCaller() {
	var buf captureBuffer
	logger := logkit.New(logkit.WithOutput(&buf), logkit.WithCaller())
	logger.Info("with caller")
	fmt.Println(strContains(buf.String(), `"caller":"`))
	// Output:
	// true
}

// Example_logLevel shows the canonical levels and min-level gating.
// Levels are the typed enum so typos fail at compile time.
func Example_logLevel() {
	var buf captureBuffer
	logger := logkit.New(logkit.WithOutput(&buf), logkit.WithMinLevel(logkit.LevelInfo))
	logger.Debug("dropped")
	logger.Info("kept")
	out := buf.String()
	fmt.Println(strContains(out, `"level":"INFO"`))
	fmt.Println(!strContains(out, `"level":"DEBUG"`))
	// Output:
	// true
	// true
}

// Example_formatVariants shows Infof / ErrorContextf / Debugf and the
// level-gated behaviour: formatting is skipped when the level is disabled.
func Example_formatVariants() {
	var buf captureBuffer
	logger := logkit.New(
		logkit.WithOutput(&buf),
		logkit.WithMinLevel(logkit.LevelInfo),
		logkit.WithContextMapper(defaultReqIDMapper),
	)
	logger.Debugf("never formatted: %d", 1)
	logger.Infof("user=%s age=%d", "alice", 30)
	logger.ErrorContextf(
		withReqID(context.Background(), "req-77"),
		"payment failed: %s",
		"timeout",
	)
	out := buf.String()
	fmt.Println(strContains(out, `"message":"user=alice age=30"`))
	fmt.Println(strContains(out, `"message":"payment failed: timeout"`))
	fmt.Println(strContains(out, `"request.id":"req-77"`))
	// Output:
	// true
	// true
	// true
}

// strContains returns true when every needle occurs in haystack. Used
// by examples that don't want to couple their expected output to the
// exact RFC3339Nano timestamp.
func strContains(haystack string, needles ...string) bool {
	for _, n := range needles {
		if !bytes.Contains([]byte(haystack), []byte(n)) {
			return false
		}
	}
	return true
}