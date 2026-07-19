package logkit_test

import (
	"context"
	"testing"
	"time"

	"github.com/hihand/go-platform/logkit"
)

// ---------- Hot-path benchmarks -------------------------------------------

// benchLogger returns a Logger writing to /dev/null-equivalent. The
// benches intentionally stay inside the logkit surface; no errkit,
// no httperr, no grpcerr. Composition cost lives in the consumer's
// own adapter and is out of scope for these measurements.
func benchLogger() logkit.Logger {
	return logkit.New(logkit.WithOutput(&discard{}))
}

func BenchmarkHotPath_NoAttrs(b *testing.B) {
	l := benchLogger()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("ok")
	}
}

func BenchmarkHotPath_String(b *testing.B) {
	l := benchLogger()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("ok",
			logkit.String(keyUserID, "u-42"),
			logkit.String(keyRequestID, "r-42"),
		)
	}
}

func BenchmarkHotPath_Mixed(b *testing.B) {
	l := benchLogger()
	base := []logkit.Attr{
		logkit.String(logkit.KeyEvent, "payment.failed"),
		logkit.String(keyPaymentID, "pay-001"),
		logkit.Int64(keyPaymentAmt, 100),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.ErrorContext(context.Background(), "downstream", base...)
	}
}

func BenchmarkHotPath_PerCall_WithCaller(b *testing.B) {
	l := logkit.New(logkit.WithOutput(&discard{}), logkit.WithCaller())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("ok")
	}
}

func BenchmarkHotPath_TimeAndDur(b *testing.B) {
	l := benchLogger()
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("ok",
			logkit.Time(keyTimeT, now),
			logkit.Dur(keyDurD, 5*time.Millisecond),
		)
	}
}

func BenchmarkHotPath_LoggerWithChainedAttrs(b *testing.B) {
	l := benchLogger().With(
		logkit.String(logkit.KeyServiceName, "payment-api"),
		logkit.String(logkit.KeyServiceVersion, "1.0.0"),
		logkit.String(logkit.KeyDeploymentEnvironment, "production"),
		logkit.String(logkit.KeyEvent, "payment.failed"),
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("ok",
			logkit.String(keyPaymentID, "pay-001"),
			logkit.Int64(keyPaymentAmt, 100),
		)
	}
}

// BenchmarkHotPath_FormatDisabled verifies the level gate skips the
// format call entirely when the level is filtered out.
func BenchmarkHotPath_FormatDisabled(b *testing.B) {
	l := logkit.New(logkit.WithOutput(&discard{}), logkit.WithMinLevel(logkit.LevelInfo))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Debugf("disabled: %d %s", i, "x")
	}
}

// BenchmarkHotPath_FormatEnabled is the typical Infof cost.
func BenchmarkHotPath_FormatEnabled(b *testing.B) {
	l := benchLogger()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Infof("user=%s age=%d", "alice", 30)
	}
}