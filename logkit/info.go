package logkit

import (
	"context"
	"fmt"
)

// Info logs at INFO level.
func (l *impl) Info(msg string, attrs ...Attr) {
	l.log(LevelInfo, nil, msg, attrs)
}

// InfoContext logs at INFO level, invoking the configured
// ContextMapper on ctx (if any) and merging the returned Attrs.
func (l *impl) InfoContext(ctx context.Context, msg string, attrs ...Attr) {
	l.log(LevelInfo, ctx, msg, attrs)
}

// Infof logs at INFO level with a printf-style message. The message
// is formatted only when the level is enabled.
func (l *impl) Infof(format string, args ...any) {
	if LevelInfo >= l.min {
		l.log(LevelInfo, nil, fmt.Sprintf(format, args...), nil)
	}
}

// InfoContextf is InfoContext with a printf-style message.
func (l *impl) InfoContextf(ctx context.Context, format string, args ...any) {
	if LevelInfo >= l.min {
		l.log(LevelInfo, ctx, fmt.Sprintf(format, args...), nil)
	}
}