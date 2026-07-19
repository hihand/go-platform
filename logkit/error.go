package logkit

import (
	"context"
	"fmt"
)

// Error logs at ERROR level.
//
// Error attributes such as error.code / error.message / error.cause
// are not built into the core logkit — the package is stdlib-only.
// Use the errkit/logkit adapter (or your own helper) to convert an
// errkit.Error into the corresponding []Attr you pass in here.
func (l *impl) Error(msg string, attrs ...Attr) {
	l.log(LevelError, nil, msg, attrs)
}

// ErrorContext logs at ERROR level, invoking the configured
// ContextMapper on ctx (if any) and merging the returned Attrs.
func (l *impl) ErrorContext(ctx context.Context, msg string, attrs ...Attr) {
	l.log(LevelError, ctx, msg, attrs)
}

// Errorf logs at ERROR level with a printf-style message. The message
// is formatted only when the level is enabled.
func (l *impl) Errorf(format string, args ...any) {
	if LevelError >= l.min {
		l.log(LevelError, nil, fmt.Sprintf(format, args...), nil)
	}
}

// ErrorContextf is ErrorContext with a printf-style message.
func (l *impl) ErrorContextf(ctx context.Context, format string, args ...any) {
	if LevelError >= l.min {
		l.log(LevelError, ctx, fmt.Sprintf(format, args...), nil)
	}
}