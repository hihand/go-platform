package errkit

import "runtime"

const defaultStackDepth = 32

// captureStack records the current goroutine's program counters with skip
// frames stripped. The returned slice is suitable for use with
// runtime.CallersFrames and matches the convention used by stdlib
// runtime.Stack.
func captureStack(skip int) []uintptr {
	pcs := make([]uintptr, defaultStackDepth)
	n := runtime.Callers(skip+1, pcs)
	return pcs[:n]
}
