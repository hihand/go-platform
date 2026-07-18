package errkit_test

import (
	"testing"

	"github.com/hihand/go-platform/errkit"
)

// TestWithStack_NonPublicEntry records frames internally but the public
// error surface is identical to New without WithStack — confirms the option
// is a no-op from the caller's perspective in v1.
func TestWithStack_NonPublicEntry(t *testing.T) {
	t.Parallel()
	a := errkit.New()
	b := errkit.New(errkit.WithStack())
	if a.Code() != b.Code() || a.Message() != b.Message() {
		t.Errorf("WithStack should not change Code/Message: a=%v b=%v", a, b)
	}
	if a.Error() != b.Error() {
		t.Errorf("WithStack should not change Error(): a=%q b=%q", a.Error(), b.Error())
	}
}
