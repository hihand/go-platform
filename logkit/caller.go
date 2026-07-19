package logkit

import (
	"runtime"
	"strconv"
	"strings"
)

// logkitFrameMarker is the path fragment used to recognise internal logkit
// frames so caller() can skip them. It lives as a single named constant
// because the same fragment also appears in the test suite; a rename here
// needs to rename it everywhere in one place.
//
// Format of the file path comes from runtime.Frame.File; it uses forward
// slashes on every platform Go supports.
const logkitFrameMarker = "logkit/"

// maxFrames bounds the runtime.Callers buffer. Eight is enough for normal
// call chains (runtime → caller → log → level method → user); a deeper
// chain still works because runtime.Callers truncates, not errors.
const maxFrames = 8

// caller captures the immediate caller of a logger method, skipping
// logkit's own frames so the user sees the actual call site.
//
// Format on the wire: "filepath:line" — never split into separate keys.
func caller(skip int) string {
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
		if !strings.Contains(frame.File, logkitFrameMarker) {
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