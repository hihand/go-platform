package logkit

import (
	"runtime"
	"strconv"
	"strings"
)

// caller captures the immediate caller of a logger method, skipping
// logkit's own frames so the user sees the actual call site.
//
// Format on the wire: "filepath:line" — never split into separate keys.
func caller(skip int) string {
	const maxFrames = 8
	pcs := [maxFrames]uintptr{}
	n := runtime.Callers(skip, pcs[:])
	if n == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pcs[:])
	for {
		frame, more := frames.Next()
		// Take the first non-logkit frame. Internal helpers (merge,
		// record build, etc.) never appear in caller output.
		if !strings.Contains(frame.File, "logkit/") {
			return trimPath(frame.File) + ":" + strconv.Itoa(frame.Line)
		}
		if !more {
			return ""
		}
	}
}

// trimPath keeps the last two segments of a file path so the caller
// field stays compact enough for grep:
//
//	payment/service.go   → "payment/service.go"
//	a/b/c/payment/svc.go → "payment/svc.go"
func trimPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 2 {
		return p
	}
	return strings.Join(parts[len(parts)-2:], "/")
}