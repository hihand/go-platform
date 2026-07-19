package logkit

import (
	"context"
	"fmt"
)

// Debug logs at DEBUG level. Skipped silently below the configured
// minimum.
func (l *impl) Debug(msg string, attrs ...Attr) {
	l.log(LevelDebug, nil, msg, attrs)
}

// DebugContext logs at DEBUG level. If the Logger was configured with
// a ContextMapper (see WithContextMapper) the supplied ctx is passed
// to it and the returned Attrs are merged into the record.
func (l *impl) DebugContext(ctx context.Context, msg string, attrs ...Attr) {
	l.log(LevelDebug, ctx, msg, attrs)
}

// Debugf logs at DEBUG level with a printf-style message. The message
// is formatted only when the level is enabled, so a disabled level
// costs no allocation.
func (l *impl) Debugf(format string, args ...any) {
	if LevelDebug >= l.min {
		l.log(LevelDebug, nil, fmt.Sprintf(format, args...), nil)
	}
}

// DebugContextf is DebugContext with a printf-style message.
func (l *impl) DebugContextf(ctx context.Context, format string, args ...any) {
	if LevelDebug >= l.min {
		l.log(LevelDebug, ctx, fmt.Sprintf(format, args...), nil)
	}
}