package configkit_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hihand/go-platform/configkit"
)

// ---------- shared example types -----------------------------------------

// ExampleAppConfig is the application-owned configuration struct. It
// belongs to this example file, not to configkit — the package has no
// knowledge of it.
type ExampleAppConfig struct {
	Server   ExampleServer   `mapstructure:"server"`
	Database ExampleDatabase `mapstructure:"database"`
	Log      ExampleLog      `mapstructure:"log"`
}

type ExampleServer struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type ExampleDatabase struct {
	URL      string `mapstructure:"url"`
	MaxConns int    `mapstructure:"max_conns"`
}

type ExampleLog struct {
	Level string `mapstructure:"level"`
}

// writeExampleYAML drops a YAML doc into a temp file and returns the
// path. Used by examples that need a config file on disk.
func writeExampleYAML(name, doc string) string {
	path := filepath.Join(os.TempDir(), name)
	_ = os.WriteFile(path, []byte(doc), 0o600)
	return path
}

// runWithEnv sets env vars for the duration of fn and restores them
// afterwards. Examples cannot use t.Setenv (no *testing.T), so this
// helper does the manual save/restore dance.
func runWithEnv(env map[string]string, fn func()) {
	saved := map[string]*string{}
	for k := range env {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = &v
		} else {
			saved[k] = nil
		}
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v == nil {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, *v)
			}
		}
	}()
	for k, v := range env {
		os.Setenv(k, v)
	}
	fn()
}

// ---------- examples ------------------------------------------------------

// ExampleNew shows the canonical wiring: a YAML file plus env-var
// overrides, with a custom key replacer so dashed names work too.
//
//	$ APP_SERVER_HOST=api.example.com myapp
//
// will pick up api.example.com even though config.yaml carries a
// different host.
func ExampleNew() {
	path := writeExampleYAML("configkit-example.yaml", `
server:
  host: 0.0.0.0
  port: 8080
database:
  url: postgres://localhost/app
  max_conns: 5
log:
  level: info
`)
	defer os.Remove(path)

	runWithEnv(map[string]string{
		"APP_SERVER_HOST": "api.example.com",
		"APP_SERVER_PORT": "9090",
	}, func() {
		var cfg ExampleAppConfig
		err := configkit.New(
			configkit.WithConfigFile(path),
			configkit.WithEnv(),
			configkit.WithEnvPrefix("APP"),
			configkit.WithValidator(func(c any) error {
				if c.(*ExampleAppConfig).Server.Port == 0 {
					return errors.New("server.port is required")
				}
				return nil
			}),
		).Load(&cfg)
		fmt.Println(err == nil)
		fmt.Println(cfg.Server.Host)
		fmt.Println(cfg.Server.Port)
	})
	// Output:
	// true
	// api.example.com
	// 9090
}

// ExampleWithConfigFile shows the option in isolation: a file with
// defaults still populates the struct when env is not enabled.
func ExampleWithConfigFile() {
	path := writeExampleYAML("configkit-file.yaml", `
server:
  host: 0.0.0.0
  port: 8080
`)
	defer os.Remove(path)

	var cfg ExampleAppConfig
	err := configkit.New(
		configkit.WithConfigFile(path),
	).Load(&cfg)
	fmt.Println(err)
	fmt.Println(cfg.Server.Host, cfg.Server.Port)
	// Output:
	// <nil>
	// 0.0.0.0 8080
}

// ExampleWithEnv demonstrates env-only loading. Without a config file
// the struct receives zero values plus any registered defaults. The
// defaults are what make Viper aware of the keys in the first place
// so the env binding can take effect during Unmarshal.
func ExampleWithEnv() {
	runWithEnv(map[string]string{
		"SERVER_HOST": "env-only.example.com",
		"LOG_LEVEL":   "debug",
	}, func() {
		var cfg ExampleAppConfig
		err := configkit.New(
			configkit.WithEnv(),
			configkit.WithDefault("server.host", "fallback.example.com"),
			configkit.WithDefault("log.level", "info"),
		).Load(&cfg)
		fmt.Println(err)
		fmt.Println(cfg.Server.Host, cfg.Log.Level)
	})
	// Output:
	// <nil>
	// env-only.example.com debug
}

// ExampleWithEnvPrefix shows how a prefix isolates this app's
// variables from anything else on the host. Without a prefix,
// generic names like SERVER_HOST leak across services.
func ExampleWithEnvPrefix() {
	runWithEnv(map[string]string{
		"SERVER_HOST":     "leaked.example.com",
		"PAYMENT_HOST":    "lost.example.com",
		"APP_SERVER_HOST": "kept.example.com",
	}, func() {
		var cfg ExampleAppConfig
		err := configkit.New(
			configkit.WithEnv(),
			configkit.WithEnvPrefix("APP"),
			configkit.WithDefault("server.host", "fallback.example.com"),
		).Load(&cfg)
		fmt.Println(err)
		fmt.Println(cfg.Server.Host)
	})
	// Output:
	// <nil>
	// kept.example.com
}

// ExampleWithEnvKeyReplacer swaps the default replacer for one that
// also folds "-" → "_", so env vars that follow dashed conventions
// (LOG_LEVEL, LOG-LEVEL, log-level) all reach the same key.
func ExampleWithEnvKeyReplacer() {
	runWithEnv(map[string]string{
		"LOG_LEVEL": "warn",
	}, func() {
		var cfg ExampleAppConfig
		err := configkit.New(
			configkit.WithEnv(),
			configkit.WithEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
			configkit.WithDefault("log.level", "info"),
		).Load(&cfg)
		fmt.Println(err)
		fmt.Println(cfg.Log.Level)
	})
	// Output:
	// <nil>
	// warn
}

// ExampleWithDefault shows defaults paving over keys the file leaves
// empty. Useful for hardening an app's boot path.
func ExampleWithDefault() {
	path := writeExampleYAML("configkit-defaults.yaml", `
server:
  host: 127.0.0.1
`)
	defer os.Remove(path)

	var cfg ExampleAppConfig
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithDefault("server.port", 4000),
		configkit.WithDefault("log.level", "warn"),
	).Load(&cfg)
	fmt.Println(err)
	fmt.Println(cfg.Server.Host, cfg.Server.Port, cfg.Log.Level)
	// Output:
	// <nil>
	// 127.0.0.1 4000 warn
}

// ExampleWithDefaults is the bulk variant of ExampleWithDefault — a
// single option registers many defaults at once. Useful when
// defaults are known up front (compile-time constants, a defaults
// struct) so the call site stays compact.
func ExampleWithDefaults() {
	path := writeExampleYAML("configkit-defaults-bulk.yaml", `
server:
  host: 127.0.0.1
`)
	defer os.Remove(path)

	var cfg ExampleAppConfig
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithDefaults(map[string]any{
			"server.host": "fallback.example.com",
			"server.port": 4000,
			"log.level":   "warn",
		}),
	).Load(&cfg)
	fmt.Println(err)
	fmt.Println(cfg.Server.Host, cfg.Server.Port, cfg.Log.Level)
	// Output:
	// <nil>
	// 127.0.0.1 4000 warn
}

// ExampleWithValidator wires a post-unmarshal validator. The
// validator runs only after Viper's Unmarshal succeeds; any non-nil
// error is surfaced verbatim.
//
// The example uses a plain closure. Real applications wire a struct
// validator such as go-playground/validator or ozzo-validation here.
func ExampleWithValidator() {
	path := writeExampleYAML("configkit-validator.yaml", `
server:
  host: 0.0.0.0
  port: 8080
`)
	defer os.Remove(path)

	required := func(c any) error {
		if c.(*ExampleAppConfig).Server.Port == 0 {
			return errors.New("server.port is required")
		}
		return nil
	}

	var cfg ExampleAppConfig
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithValidator(required),
	).Load(&cfg)
	fmt.Println(err)
	fmt.Println(cfg.Server.Port)
	// Output:
	// <nil>
	// 8080
}

// ExampleLoader_Load_propagatesValidationErrors shows that a
// validator returning a non-nil error reaches the caller untouched.
func ExampleLoader_Load_propagatesValidationErrors() {
	err := configkit.New(
		configkit.WithValidator(func(any) error { return errors.New("nope") }),
	).Load(new(ExampleAppConfig))
	fmt.Println(err)
	// Output:
	// nope
}
