package logkit

import (
	"context"
	"slices"
	"sync"
	"time"
)

// bufPool amortises the per-call scratch buffer cost across log calls.
// Buffer reuse is safe here because each emit() consumes the buffer
// fully before the next caller checks out.
var bufPool = sync.Pool{
	New: func() any { b := make([]byte, 0, 512); return &b },
}

// log is the internal sink shared by every level method on Logger.
//
// Precedence (lowest → highest; later wins on duplicate keys because
// the JSON decoder keeps the last occurrence):
//
//	static → withAttrs → ctxAttrs → callAttrs
//
// When the Logger has no mapper (the common case) the ctx argument is
// ignored and the non-context path is allocation-free.
func (l *impl) log(lvl Level, ctx context.Context, msg string, attrs []Attr) {
	if lvl < l.min {
		return
	}

	bufp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufp)
	buf := *bufp

	var callerStr string
	if l.withCaller {
		// 5 = runtime.Callers → caller → log → level method → user
		callerStr = caller(5)
	}

	// Lift KeyEvent into the top-level "event" schema field. Call
	// attrs win over With attrs.
	event := findEvent(attrs)
	if event == "" {
		event = findEvent(l.withAttrs)
	}

	// Build the merged attr list. ctxAttrs is nil when no mapper is
	// configured, so the slices.Concat call below is a no-op there.
	ctxAttrs := []Attr(nil)
	if l.mapper != nil && ctx != nil {
		ctxAttrs = l.mapper(ctx, nil)
	}

	merged := slices.Concat(l.withAttrs, ctxAttrs, attrs)
	if event != "" {
		merged = stripEvent(merged)
	}

	r := Record{
		ts:     time.Now(),
		level:  lvl,
		msg:    msg,
		event:  event,
		caller: callerStr,
		static: l.static,
		attrs:  merged,
	}

	newBuf, _ := l.encoder.emit(l.out, buf, r)
	*bufp = newBuf
}

// findEvent returns the value of the first KeyEvent attr in attrs, or
// "" if none. The kind check is in here so callers don't have to know
// how Attr is encoded.
func findEvent(attrs []Attr) string {
	for _, a := range attrs {
		if a.kind == attrString && a.key == KeyEvent {
			return a.str
		}
	}
	return ""
}

// stripEvent drops the KeyEvent attr from a slice (after its value has
// been lifted into the schema field). The order of all other attrs is
// preserved.
func stripEvent(attrs []Attr) []Attr {
	for i, a := range attrs {
		if a.kind == attrString && a.key == KeyEvent {
			return slices.Delete(attrs, i, i+1)
		}
	}
	return attrs
}