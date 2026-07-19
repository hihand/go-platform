package logkit

import "io"

// Option mutates the impl produced by New. Options are applied in the
// order they are supplied; later options override earlier ones for
// scalar fields.
type Option func(*impl)

// WithOutput sets the writer the Logger emits to. The default is
// os.Stdout. The Logger does not own the writer — closing it is the
// caller's responsibility.
func WithOutput(w io.Writer) Option {
	return func(l *impl) {
		if w == nil {
			return
		}
		l.out = w
	}
}

// WithMinLevel sets the minimum level the Logger emits. Logs below
// min are dropped silently. The default is INFO.
//
// The argument is the typed Level enum from enum.go so typos are
// caught at compile time. Unknown values are clamped to INFO.
func WithMinLevel(lvl Level) Option {
	return func(l *impl) {
		l.min = clampLevel(lvl)
	}
}

// WithService attaches service.name + service.version as static fields.
// Both default to empty when omitted.
func WithService(name, version string) Option {
	return func(l *impl) {
		if name != "" {
			l.static = append(l.static, String(KeyServiceName, name))
		}
		if version != "" {
			l.static = append(l.static, String(KeyServiceVersion, version))
		}
	}
}

// WithDeployment attaches deployment.environment as a static field.
func WithDeployment(env string) Option {
	return func(l *impl) {
		if env != "" {
			l.static = append(l.static, String(KeyDeploymentEnvironment, env))
		}
	}
}

// WithCaller enables caller capture on every log. Caller is added as a
// single "filepath:line" field — never split into separate keys. Off
// by default.
func WithCaller() Option {
	return func(l *impl) {
		l.withCaller = true
	}
}

// WithStatic attaches an arbitrary static key/value pair. Used for
// fixed metadata that should appear on every log record (cluster,
// region, build SHA, ...). Pass any Key — typically a canonical
// constant, or AnyKey for a one-off name.
func WithStatic(key Key, value string) Option {
	return func(l *impl) {
		if key == "" {
			return
		}
		l.static = append(l.static, String(key, value))
	}
}