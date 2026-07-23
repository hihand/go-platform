package configkit

import "strings"

// impl is the concrete Loader. It is unexported; callers only ever
// see the Loader interface. The struct is intentionally flat — every
// field maps to a single option, with no derived state. Load() builds
// a fresh Viper instance on every call from these fields, so the
// Loader is safe to share across goroutines and re-invoke after
// mutating the configuration sources.
type impl struct {
	file      string            // path to the YAML config file ("" → none)
	env       bool              // enable env-var binding
	prefix    string            // env var prefix ("" → none)
	replacer  *strings.Replacer // env key normaliser (nil → "." replaced with "_")
	defaults  map[string]any    // registered defaults, applied before the file
	validator func(any) error   // post-unmarshal validator (nil → skip)
}

// New constructs a Loader with the supplied options applied in order.
// A zero-value call returns a Loader that:
//
//   - has no config file,
//   - does not read env vars,
//   - has no defaults,
//   - has no validator.
//
// Such a Loader still works — Load() simply unmarshals an empty
// Viper instance into cfg. Callers that need file/env/defaults wire
// them in via the With* options.
func New(opts ...Option) Loader {
	l := &impl{
		replacer: strings.NewReplacer(".", "_"),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}
