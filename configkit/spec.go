// Package configkit is a thin, idiomatic wrapper around Viper that loads
// configuration from a YAML file and environment variables, then
// unmarshals the result into a caller-defined struct.
//
// # Design
//
//	configkit.New(opts ...Option) configkit.Loader
//	loader.Load(cfg any) error
//
// The package owns three things and nothing else:
//
//  1. the loading mechanics (file + env precedence),
//  2. unmarshalling into user-defined structs,
//  3. an optional final validation step.
//
// The package never inspects the struct that callers pass to Load —
// AppConfig, DatabaseConfig, JWTConfig and the rest belong to the
// application. configkit does not import Viper in any user-visible
// type or signature, so swapping the underlying library is a
// single-file change.
//
// # Loading flow
//
// Load() is the single entry point. It creates a fresh Viper instance
// on every call, so a Loader is goroutine-safe and re-runnable:
//
//	v := configkit.New(
//	    configkit.WithConfigFile("config.yaml"),
//	    configkit.WithEnv(),
//	    configkit.WithEnvPrefix("APP"),
//	    configkit.WithValidator(func(c any) error { ... }),
//	)
//
//	var cfg Config
//	if err := v.Load(&cfg); err != nil { ... }
//
// Precedence inside Load, last source wins:
//
//	defaults → config file → environment
//
// Environment variables override values from the YAML file by
// default. The Viper key — e.g. "server.port" — and the env var —
// e.g. APP_SERVER_PORT — are related via the EnvKeyReplacer.
//
// # Layout
//
//	spec.go     — Loader interface
//	new.go      — impl + New()
//	option.go   — functional options (WithConfigFile, WithEnv, ...)
//	load.go     — Load() implementation
//	common.go   — small unexported helpers shared between files
//	configkit_test.go   — package tests, consolidated
//	example_test.go     — runnable godoc examples
package configkit

// Loader loads configuration from configured sources and fills a
// caller-defined struct. A Loader is constructed by New and is safe
// to call Load on repeatedly; each call rebuilds its Viper instance
// from the configured sources so callers can re-load after editing
// the config file or rotating environment variables.
type Loader interface {
	// Load reads configuration from the configured sources and
	// unmarshals it into cfg. cfg must be a non-nil pointer to a
	// struct (or any value that Viper's Unmarshal accepts).
	//
	// If a validator was supplied via WithValidator it runs only
	// after a successful Unmarshal, receiving the same cfg value.
	// A validator that returns a non-nil error is returned to the
	// caller verbatim.
	//
	// Common returns:
	//   - *viper.ConfigFileNotFoundError-equivalent: never returned.
	//     A missing config file is silently ignored so an app can
	//     boot from env vars alone.
	//   - any other unmarshal / I/O error from Viper, untouched.
	Load(cfg any) error
}
