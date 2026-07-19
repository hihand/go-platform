// Package logkit is a small, predictable, stdlib-only structured
// (JSON) logger for Go. It targets the same niche as log/slog or
// zap's core: one Logger per process, derived loggers via With,
// context-aware correlation via a single ContextMapper seam, and a
// hand-rolled JSON encoder on a pooled buffer to keep the hot path
// allocation-light.
//
// # Design
//
//	logkit.New(opts ...Option) Logger
//
// Every method on Logger flows through a single internal hot path
// (log.go) which:
//
//   - gates by the configured minimum level (typed Level enum),
//   - optionally captures caller information,
//   - merges attrs in the precedence order
//     static < withAttrs < ctxAttrs < callAttrs
//     (later wins on duplicate keys),
//   - hoists the KeyEvent string into a top-level schema field,
//   - hands the result to a stateless encoder which writes one JSON
//     object per call to the configured io.Writer.
//
// Field names are the typed Key enum from enum.go. Canonical names
// (event, service.name, ...) are typed constants; everything else
// uses the AnyKey escape hatch so call sites still get type checking.
//
// The ContextMapper is the only seam between the logger and
// context.Context. Without one, the context is ignored entirely and
// the non-context methods stay allocation-free. Mappers use an
// append-style signature so chaining and capacity reuse are trivial.
//
// # Layout
//
//	spec.go     — Logger interface
//	enum.go     — Level + Key typed enums
//	new.go      — impl + New()
//	options.go  — functional options
//	mapper.go   — ContextMapper + WithContextMapper
//	attr.go     — Attr tagged union + typed constructors
//	log.go      — internal hot path
//	record.go   — internal value passed to the encoder
//	encode.go   — JSON encoder
//	caller.go   — caller capture
//
// Formatted variants (*f / *Contextf) follow fmt.Sprintf semantics.
// Formatting is skipped when the level is disabled, so the cost of a
// gated Debugf call is one branch and one variadic slice header.
package logkit

import (
	"io"
	"os"
)

// impl is the concrete Logger implementation. It is unexported; callers
// only see the Logger interface. The struct is deliberately flat to
// keep the hot path branch-free.
type impl struct {
	out        io.Writer
	encoder    encoder
	min        Level
	mapper     ContextMapper // optional — see mapper.go
	static     []Attr        // service.* + deployment.* — set once per Logger
	withAttrs  []Attr        // accumulated via .With()
	withCaller bool          // when true, caller info is captured from runtime
}

// New constructs a Logger with the supplied options applied in order.
// The default sink is os.Stdout, the default minimum level is INFO,
// caller capture is off, and no ContextMapper is configured (context
// is ignored).
//
// A zero-value New() is legal — it returns a valid Logger writing JSON
// to stdout at INFO with no static fields set.
func New(opts ...Option) Logger {
	l := &impl{
		out:     os.Stdout,
		min:     LevelInfo,
		encoder: defaultEncoder(),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}