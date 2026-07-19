package logkit

import (
	"context"
	"fmt"
)

// Warn logs at WARN level.
func (l *impl) Warn(msg string, attrs ...Attr) {
	l.log(LevelWarn, nil, msg, attrs)
}

// WarnContext logs at WARN level, invoking the configured
// ContextMapper on ctx (if any) and merging the returned Attrs.
func (l *impl) WarnContext(ctx context.Context, msg string, attrs ...Attr) {
	l.log(LevelWarn, ctx, msg, attrs)
}

// Warnf logs at WARN level with a printf-style message. The message
// is formatted only when the level is enabled.
func (l *impl) Warnf(format string, args ...any) {
	if LevelWarn >= l.min {
		l.log(LevelWarn, nil, fmt.Sprintf(format, args...), nil)
	}
}

// WarnContextf is WarnContext with a printf-style message.
func (l *impl) WarnContextf(ctx context.Context, format string, args ...any) {
	if LevelWarn >= l.min {
		l.log(LevelWarn, ctx, fmt.Sprintf(format, args...), nil)
	}
}