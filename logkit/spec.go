package logkit

import "context"

// Logger is the interface implemented by every logkit logger. It is
// intentionally minimal: four levels, four *Context variants, four
// formatted variants, four *Context formatted variants, and With
// for derived loggers. Context is read-only from the Logger's
// perspective; the configured ContextMapper decides what to extract.
//
// Formatted variants (*f) follow fmt.Sprintf semantics. They share
// the level gate with the plain variants: the message is formatted
// only if the level is enabled, so the format+args cost is avoided
// when the record would be dropped.
type Logger interface {
	// Debug logs at DEBUG level.
	Debug(msg string, attrs ...Attr)
	// Info logs at INFO level.
	Info(msg string, attrs ...Attr)
	// Warn logs at WARN level.
	Warn(msg string, attrs ...Attr)
	// Error logs at ERROR level.
	Error(msg string, attrs ...Attr)

	// DebugContext / InfoContext / WarnContext / ErrorContext log at
	// the named level while invoking the Logger's configured
	// ContextMapper on ctx (if any) and merging the returned Attrs
	// into the record.
	DebugContext(ctx context.Context, msg string, attrs ...Attr)
	InfoContext(ctx context.Context, msg string, attrs ...Attr)
	WarnContext(ctx context.Context, msg string, attrs ...Attr)
	ErrorContext(ctx context.Context, msg string, attrs ...Attr)

	// Debugf / Infof / Warnf / Errorf log at the named level with a
	// printf-style message. Formatting is skipped when the level is
	// disabled.
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)

	// DebugContextf / InfoContextf / WarnContextf / ErrorContextf
	// combine the Context and formatted behaviour.
	DebugContextf(ctx context.Context, format string, args ...any)
	InfoContextf(ctx context.Context, format string, args ...any)
	WarnContextf(ctx context.Context, format string, args ...any)
	ErrorContextf(ctx context.Context, format string, args ...any)

	// With returns a child logger with attrs permanently attached.
	// Caller-merge precedence applies: child attrs override parent attrs.
	With(attrs ...Attr) Logger
}