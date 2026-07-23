package configkit

import "strings"

// Option mutates the Loader produced by New. Options are applied in
// the order supplied; later options override earlier ones for
// scalar fields (prefix, replacer, validator, file path).
type Option func(*impl)

// WithConfigFile sets the path to the YAML configuration file. The
// file is read by Load() via Viper's ReadInConfig. Either flavor of
// "missing" error — viper.ConfigFileNotFoundError from the discovery
// path, or os.IsNotExist from an explicit SetConfigFile path — is
// swallowed so an application can boot from env vars alone.
//
// An empty path is a no-op so callers can forward user-supplied
// paths without a guard.
func WithConfigFile(path string) Option {
	return func(l *impl) {
		if path == "" {
			return
		}
		l.file = path
	}
}

// WithEnv enables binding of every environment variable into Viper.
// Without this option, env vars are ignored regardless of any prefix
// or key replacer that may be configured.
//
// Call this once before WithEnvPrefix / WithEnvKeyReplacer — the
// order of options only matters when the same option is repeated.
func WithEnv() Option {
	return func(l *impl) {
		l.env = true
	}
}

// WithEnvPrefix sets the prefix that env vars must carry to be
// visible to Viper. Empty prefix disables the prefix.
//
//	WithEnvPrefix("APP"), env var APP_SERVER_PORT → key "server.port"
//
// The prefix is applied by Viper after WithEnvKeyReplacer; both are
// independent knobs.
func WithEnvPrefix(prefix string) Option {
	return func(l *impl) {
		l.prefix = prefix
	}
}

// WithEnvKeyReplacer sets the replacer used to normalise env-var
// names into Viper keys. The default is strings.NewReplacer(".", "_")
// so "server.port" maps to env var SERVER_PORT.
//
// Pass nil to keep the default; pass your own *strings.Replacer to
// customise, e.g. strings.NewReplacer(".", "_", "-", "_") to also
// accept dashed names.
func WithEnvKeyReplacer(r *strings.Replacer) Option {
	return func(l *impl) {
		if r == nil {
			return
		}
		l.replacer = r
	}
}

// WithDefault registers a single fallback value for the given key.
// Multiple calls register multiple defaults; precedence inside Load()
// is defaults < config file < environment, so a default is masked
// whenever the file or the environment supplies a value for the
// same key.
//
//	Key is a dotted Viper path (e.g. "server.port"). value is any
//	value that Viper's Set accepts.
func WithDefault(key string, value any) Option {
	return func(l *impl) {
		if key == "" {
			return
		}
		if l.defaults == nil {
			l.defaults = make(map[string]any)
		}
		l.defaults[key] = value
	}
}

// WithDefaults registers many defaults at once. It is the bulk
// variant of WithDefault; each entry follows the same precedence
// rules (defaults < config file < environment) and the same dotted
// key convention.
//
// A nil or empty map is a no-op. Empty keys inside the map are
// skipped silently so callers can forward user-supplied maps
// without a guard.
//
// Use WithDefault when adding one key at a time inside a
// conditional; use WithDefaults when the defaults are known up
// front (e.g. compile-time constants, a defaults struct).
func WithDefaults(m map[string]any) Option {
	return func(l *impl) {
		if len(m) == 0 {
			return
		}
		if l.defaults == nil {
			l.defaults = make(map[string]any, len(m))
		}
		for key, value := range m {
			if key == "" {
				continue
			}
			l.defaults[key] = value
		}
	}
}

// WithValidator installs a post-unmarshal validator. The validator
// runs only after Viper's Unmarshal has succeeded and receives the
// same value the caller passed to Load().
//
// Typical wiring with a struct-validator library:
//
//	configkit.WithValidator(func(c any) error {
//	    return validate.Struct(c)
//	})
//
// Pass nil to clear any previously installed validator. A validator
// that returns a non-nil error is returned to the caller verbatim —
// no wrapping, no translation.
func WithValidator(fn func(any) error) Option {
	return func(l *impl) {
		l.validator = fn
	}
}
