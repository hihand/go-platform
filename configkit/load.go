package configkit

import (
	"errors"
	"os"

	"github.com/spf13/viper"
)

// Load reads configuration from the configured sources and fills cfg.
// Per the Loader contract a new Viper instance is built on every
// call, so callers can re-load after editing the config file or
// rotating env vars without rebuilding the Loader.
//
// The flow matches the spec:
//
//  1. create Viper,
//  2. load the config file (if configured),
//  3. ignore viper.ConfigFileNotFoundError,
//  4. bind env (if configured),
//  5. set EnvKeyReplacer,
//  6. set EnvPrefix,
//  7. register defaults,
//  8. Unmarshal into cfg,
//  9. run validator (if configured) only on Unmarshal success.
//
// Any error before Unmarshal surfaces directly to the caller.
func (l *impl) Load(cfg any) error {
	v := viper.New()

	if err := l.loadFile(v); err != nil {
		return err
	}
	if l.env {
		v.SetEnvKeyReplacer(l.replacer)
		if l.prefix != "" {
			v.SetEnvPrefix(l.prefix)
		}
		v.AutomaticEnv()
	}
	l.applyDefaults(v)

	if err := v.Unmarshal(cfg); err != nil {
		return err
	}
	if l.validator != nil {
		return l.validator(cfg)
	}
	return nil
}

// loadFile reads the YAML config file when one is configured. A
// missing file is not an error: applications can boot from defaults
// and environment variables alone, so the caller never has to
// distinguish "file missing" from "no file configured".
//
// Viper emits viper.ConfigFileNotFoundError from its search-path
// discovery path only; when a file path is set explicitly via
// SetConfigFile, ReadInConfig surfaces an os.ErrNotExist-style error
// instead. Both shapes are caught here so callers get the same
// behaviour either way.
//
// Any other I/O or parse failure is returned verbatim because it
// points at a real problem the caller must see.
func (l *impl) loadFile(v *viper.Viper) error {
	if l.file == "" {
		return nil
	}
	v.SetConfigFile(l.file)
	v.SetConfigType(configFileType)
	if err := v.ReadInConfig(); err != nil {
		if isMissingConfigFile(err) {
			return nil
		}
		return err
	}
	return nil
}

// isMissingConfigFile reports whether err is Viper's
// ConfigFileNotFoundError or any os.IsNotExist error wrapped around
// it. Viper returns the former only when discovering by name and
// the latter when a file path is set explicitly.
func isMissingConfigFile(err error) bool {
	var notFound viper.ConfigFileNotFoundError
	if errors.As(err, &notFound) {
		return true
	}
	return os.IsNotExist(err)
}

// applyDefaults registers every key/value pair added via
// WithDefault. Order is irrelevant — Viper's precedence (defaults
// < file < env) means later registrations for the same key cannot
// shadow earlier ones anyway.
func (l *impl) applyDefaults(v *viper.Viper) {
	for key, value := range l.defaults {
		v.SetDefault(key, value)
	}
}
