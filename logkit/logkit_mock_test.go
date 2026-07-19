package logkit_test

import (
	"bytes"

	"github.com/hihand/go-platform/logkit"
)

// captureBuffer is the shared in-memory sink used by examples. The
// implementation is deliberately trivial; the package's promise is that
// any io.Writer works as a Logger sink. bytes.Buffer is embedded by
// type only so the underlying type's methods (Write, String) are
// promoted directly onto *captureBuffer.
type captureBuffer struct {
	bytes.Buffer
}

// Shared test/bench keys. Canonical schema constants live in
// logkit.Key*; application-level correlation/business fields below
// are typed via logkit.AnyKey. Centralising them here avoids
// redeclaration across the test, bench and example files while
// keeping every call site typed.
var (
	keyRequestID  = logkit.AnyKey("request.id")
	keyPaymentID  = logkit.AnyKey("payment.id")
	keyPaymentAmt = logkit.AnyKey("payment.amount")
	keyK          = logkit.AnyKey("k")
	keyA          = logkit.AnyKey("a")
	keyB          = logkit.AnyKey("b")
	keyUserID     = logkit.AnyKey("user.id")
	keyTimeT      = logkit.AnyKey("t")
	keyDurD       = logkit.AnyKey("d")
)

// discard is a sink that throws bytes away — used by every bench so
// the measurement reflects the log path, not the writer.
type discard struct{ n int64 }

func (d *discard) Write(p []byte) (int, error) {
	d.n += int64(len(p))
	return len(p), nil
}