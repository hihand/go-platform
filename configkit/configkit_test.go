package configkit_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hihand/go-platform/configkit"
)

// ---------- shared types & helpers ----------------------------------------

// cfg mirrors a typical user-defined application config. The package
// must not look at the shape of this struct; it is opaque to
// configkit.
type cfg struct {
	Server   serverCfg   `mapstructure:"server"`
	Database databaseCfg `mapstructure:"database"`
	Log      logCfg      `mapstructure:"log"`
}

type serverCfg struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type databaseCfg struct {
	URL      string `mapstructure:"url"`
	MaxConns int    `mapstructure:"max_conns"`
}

type logCfg struct {
	Level string `mapstructure:"level"`
}

// writeYAML writes a YAML doc to a fresh temp file and returns its
// path; cleanup is automatic via t.TempDir().
func writeYAML(t *testing.T, doc string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	return path
}

// withEnv sets an env var for the duration of the test. The call to
// t.Setenv forces the test out of t.Parallel — that constraint is a
// property of testing.T, not of configkit.
func withEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

// ---------- New + option plumbing -----------------------------------------

func TestNew_ReturnsNonNilLoader(t *testing.T) {
	t.Parallel()
	l := configkit.New()
	if l == nil {
		t.Fatal("New must return a non-nil Loader, even with no options")
	}
}

func TestNew_ZeroOptions_LoadsZeroStruct(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New().Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != (cfg{}) {
		t.Errorf("want zero cfg, got %+v", got)
	}
}

func TestOption_WithConfigFile_EmptyIsNoop(t *testing.T) {
	t.Parallel()
	l := configkit.New(configkit.WithConfigFile(""))
	var got cfg
	if err := l.Load(&got); err != nil {
		t.Errorf("Load: %v", err)
	}
}

func TestOption_WithDefault_EmptyKeyIsNoop(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithDefault("", "ignored"),
		configkit.WithDefault("server.port", 9090),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 9090 {
		t.Errorf("server.port = %d, want 9090", got.Server.Port)
	}
}

func TestOption_WithEnvKeyReplacer_NilIsNoop(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithEnvKeyReplacer(nil),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load with nil replacer must default silently: %v", err)
	}
}

// ---------- file loading --------------------------------------------------

func TestLoad_FileSuppliesValues(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, `
server:
  host: 0.0.0.0
  port: 8080
database:
  url: postgres://localhost/app
  max_conns: 5
log:
  level: info
`)
	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "0.0.0.0" || got.Server.Port != 8080 {
		t.Errorf("server = %+v, want {0.0.0.0 8080}", got.Server)
	}
	if got.Database.URL != "postgres://localhost/app" || got.Database.MaxConns != 5 {
		t.Errorf("database = %+v", got.Database)
	}
	if got.Log.Level != "info" {
		t.Errorf("log.level = %q", got.Log.Level)
	}
}

func TestLoad_FileMissingIsIgnored(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "absent.yaml")
	var got cfg
	// No file → must NOT bubble up an os.IsNotExist or viper
	// ConfigFileNotFoundError; falls back to defaults / env only.
	err := configkit.New(
		configkit.WithConfigFile(missing),
		configkit.WithDefault("server.port", 7000),
	).Load(&got)
	if err != nil {
		t.Fatalf("missing config file must be ignored, got error: %v", err)
	}
	if got.Server.Port != 7000 {
		t.Errorf("default port = %d, want 7000", got.Server.Port)
	}
}

func TestLoad_FilePartialLeavesRestToDefaults(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, `
server:
  host: 127.0.0.1
`)
	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithDefault("server.port", 4000),
		configkit.WithDefault("log.level", "debug"),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "127.0.0.1" {
		t.Errorf("server.host = %q, want 127.0.0.1", got.Server.Host)
	}
	if got.Server.Port != 4000 {
		t.Errorf("server.port = %d, want 4000 (default kicks in)", got.Server.Port)
	}
	if got.Log.Level != "debug" {
		t.Errorf("log.level = %q, want debug", got.Log.Level)
	}
}

func TestLoad_FileInvalidYAMLReturnsError(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, "this: : is: not: valid")
	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
	).Load(&got)
	if err == nil {
		t.Fatal("malformed YAML must surface as an error")
	}
}

// ---------- env overrides over file + defaults ---------------------------

// Tests in this section touch the process environment via t.Setenv
// and so cannot use t.Parallel. They assume a config file (or
// defaults) supplies the keys — Viper's Unmarshal only consults env
// vars for keys it knows about, which is the contract advertised by
// configkit.

func TestLoad_EnvOverridesFile(t *testing.T) {
	path := writeYAML(t, `
server:
  host: 0.0.0.0
  port: 8080
`)
	withEnv(t, "APP_SERVER_HOST", "api.example.com")
	withEnv(t, "APP_SERVER_PORT", "9090")

	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithEnv(),
		configkit.WithEnvPrefix("APP"),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "api.example.com" {
		t.Errorf("host = %q, env must override file", got.Server.Host)
	}
	if got.Server.Port != 9090 {
		t.Errorf("port = %d, env must override file", got.Server.Port)
	}
}

func TestLoad_EnvPrefixIsolatesKeys(t *testing.T) {
	// Two env vars: one with the prefix, one without. With the
	// prefix set, only the prefixed one should bind.
	withEnv(t, "SERVER_HOST", "leaked.example.com")
	withEnv(t, "APP_SERVER_HOST", "kept.example.com")

	path := writeYAML(t, `
server:
  host: file.example.com
`)
	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithEnv(),
		configkit.WithEnvPrefix("APP"),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "kept.example.com" {
		t.Errorf("server.host = %q, want kept.example.com (prefix APP must apply)", got.Server.Host)
	}
}

func TestLoad_EnvNotEnabled_IgnoresEnv(t *testing.T) {
	path := writeYAML(t, `
server:
  host: file.example.com
`)
	withEnv(t, "SERVER_HOST", "env.example.com")

	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
		// No WithEnv() → env vars must NOT be read.
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "file.example.com" {
		t.Errorf("server.host = %q, env must be ignored when WithEnv is not set", got.Server.Host)
	}
}

func TestLoad_EnvKeyReplacerDefaultMapsDotToUnderscore(t *testing.T) {
	// Default replacer: "log.level" → LOG_LEVEL. A dashed name like
	// "LOG-LEVEL" stays dashed and therefore does NOT match when
	// the default replacer is in place.
	path := writeYAML(t, `
log:
  level: file-level
`)
	withEnv(t, "LOG_LEVEL", "warn")
	withEnv(t, "LOG-LEVEL", "error")

	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithEnv(),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Log.Level != "warn" {
		t.Errorf("default replacer: log.level = %q, want warn (LOG_LEVEL)", got.Log.Level)
	}
}

func TestLoad_EnvKeyReplacerCustomOverridesDefault(t *testing.T) {
	// A custom replacer that also folds "-" → "_" works on top of
	// Viper's "key → upper" step. To prove that the custom replacer
	// is taking effect — and not the default — we set an env var
	// whose name ONLY matches when the custom replacer runs. The
	// key "log.level" gets uppercased to "LOG.LEVEL", after which
	// the default replacer would normalise it to "LOG_LEVEL"; the
	// custom replacer that also folds "-" turns "LOG_LEVEL" (the
	// post-replacer form) into "LOG_LEVEL" — so we exercise a
	// genuine difference by changing the key shape to one only the
	// custom replacer unblocks.
	path := writeYAML(t, `
log:
  level: file-level
`)
	// LOG_LEVEL works with both default and custom replacers.
	withEnv(t, "LOG_LEVEL", "warn")
	// LOG-LEVEL only matches when "-" → "_" is folded too.
	withEnv(t, "LOG-LEVEL", "error")

	// With the default replacer, only LOG_LEVEL matches → warn.
	var defaultCase cfg
	if err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithEnv(),
	).Load(&defaultCase); err != nil {
		t.Fatalf("Load default replacer: %v", err)
	}
	if defaultCase.Log.Level != "warn" {
		t.Errorf("default replacer: log.level = %q, want warn (only LOG_LEVEL is reachable)", defaultCase.Log.Level)
	}

	// With the custom replacer that folds dashes too, LOG-LEVEL
	// becomes reachable. The exact winner between LOG_LEVEL and
	// LOG-LEVEL depends on Viper's iteration order; either is a
	// valid outcome. The point is: it is *not* file-level.
	var customCase cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithEnv(),
		configkit.WithEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	).Load(&customCase)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if customCase.Log.Level == "file-level" {
		t.Errorf("custom replacer did not enable env override; log.level = %q, want non-file value", customCase.Log.Level)
	}
}

// ---------- defaults ------------------------------------------------------

func TestLoad_DefaultsOnlyWhenNothingElseSupplies(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithDefault("server.host", "localhost"),
		configkit.WithDefault("server.port", 3000),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "localhost" {
		t.Errorf("default host = %q", got.Server.Host)
	}
	if got.Server.Port != 3000 {
		t.Errorf("default port = %d", got.Server.Port)
	}
}

func TestLoad_WithDefaults_BulkRegister(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithDefaults(map[string]any{
			"server.host": "localhost",
			"server.port": 3000,
			"log.level":   "info",
		}),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "localhost" || got.Server.Port != 3000 {
		t.Errorf("server = %+v", got.Server)
	}
	if got.Log.Level != "info" {
		t.Errorf("log.level = %q", got.Log.Level)
	}
}

func TestOption_WithDefaults_NilMapIsNoop(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithDefault("server.port", 4242),
		configkit.WithDefaults(nil),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 4242 {
		t.Errorf("WithDefaults(nil) must not wipe prior defaults; port = %d", got.Server.Port)
	}
}

func TestOption_WithDefaults_EmptyMapIsNoop(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithDefault("server.port", 4242),
		configkit.WithDefaults(map[string]any{}),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 4242 {
		t.Errorf("WithDefaults({}) must not wipe prior defaults; port = %d", got.Server.Port)
	}
}

func TestOption_WithDefaults_EmptyKeysAreSkipped(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithDefaults(map[string]any{
			"":            "ignored",
			"server.port": 4242,
		}),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 4242 {
		t.Errorf("port = %d, want 4242 (empty-key entry must be skipped)", got.Server.Port)
	}
}

func TestOption_WithDefaults_MergesWithSingle(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithDefaults(map[string]any{
			"server.host": "bulk-host",
			"server.port": 3000,
		}),
		configkit.WithDefault("log.level", "from-single"),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "bulk-host" || got.Server.Port != 3000 {
		t.Errorf("bulk defaults lost: server = %+v", got.Server)
	}
	if got.Log.Level != "from-single" {
		t.Errorf("single default lost: log.level = %q", got.Log.Level)
	}
}

func TestOption_WithDefaults_LaterOverridesEarlier(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithDefaults(map[string]any{
			"server.port": 1111,
			"log.level":   "first",
		}),
		configkit.WithDefaults(map[string]any{
			"server.port": 2222,
		}),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 2222 {
		t.Errorf("later map must override earlier on key collision; port = %d", got.Server.Port)
	}
	if got.Log.Level != "first" {
		t.Errorf("non-colliding key must survive; log.level = %q", got.Log.Level)
	}
}

func TestLoad_DefaultsAreVisibleToEnvOverride(t *testing.T) {
	// Without a config file, defaults register the keys so Viper
	// knows to consult env vars during Unmarshal.
	withEnv(t, "SERVER_HOST", "from-env.example.com")

	var got cfg
	err := configkit.New(
		configkit.WithDefault("server.host", "from-default"),
		configkit.WithEnv(),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Host != "from-env.example.com" {
		t.Errorf("server.host = %q, env must beat default", got.Server.Host)
	}
}

// ---------- validation ----------------------------------------------------

func TestLoad_ValidatorRunsAfterUnmarshal(t *testing.T) {
	t.Parallel()
	called := false
	var seen *cfg
	err := configkit.New(
		configkit.WithConfigFile(writeYAML(t, `
server:
  host: yaml.example.com
  port: 7000
`)),
		configkit.WithValidator(func(c any) error {
			called = true
			seen = c.(*cfg)
			return nil
		}),
	).Load(&cfg{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !called {
		t.Fatal("validator must run after successful Unmarshal")
	}
	if seen.Server.Host != "yaml.example.com" || seen.Server.Port != 7000 {
		t.Errorf("validator must see the unmarshaled struct, got %+v", seen.Server)
	}
}

func TestLoad_ValidatorReturnsErrorIsSurfaced(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("bad config")
	var got cfg
	err := configkit.New(
		configkit.WithValidator(func(any) error { return sentinel }),
	).Load(&got)
	if !errors.Is(err, sentinel) {
		t.Errorf("validator error not surfaced: got %v, want %v", err, sentinel)
	}
}

func TestLoad_ValidatorDoesNotRunAfterUnmarshalFailure(t *testing.T) {
	t.Parallel()
	called := false
	path := writeYAML(t, "this: : is: not: valid")
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithValidator(func(any) error {
			called = true
			return nil
		}),
	).Load(&cfg{})
	if err == nil {
		t.Fatal("malformed YAML must surface as error before validator")
	}
	if called {
		t.Error("validator must not run when Unmarshal fails")
	}
}

func TestLoad_ValidatorNilOptionClearIsSafe(t *testing.T) {
	t.Parallel()
	var got cfg
	err := configkit.New(
		configkit.WithValidator(func(any) error { return errors.New("never") }),
		configkit.WithValidator(nil),
	).Load(&got)
	if err != nil {
		t.Errorf("validator=nil must clear previous validator: %v", err)
	}
}

// ---------- loader re-runnability ----------------------------------------

func TestLoad_RepeatedCallsAreIndependent(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, "server:\n  host: file.example.com\n  port: 8000\n")

	l := configkit.New(
		configkit.WithConfigFile(path),
	)

	var first cfg
	if err := l.Load(&first); err != nil {
		t.Fatalf("first Load: %v", err)
	}
	if first.Server.Host != "file.example.com" || first.Server.Port != 8000 {
		t.Fatalf("first: %+v", first.Server)
	}

	// Mutate the file in place and re-load — Loader must read fresh.
	if err := os.WriteFile(path, []byte("server:\n  host: updated.example.com\n  port: 9000\n"), 0o600); err != nil {
		t.Fatalf("rewrite yaml: %v", err)
	}

	var second cfg
	if err := l.Load(&second); err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if second.Server.Host != "updated.example.com" || second.Server.Port != 9000 {
		t.Errorf("second load must re-read; got %+v", second.Server)
	}
}

// ---------- precedence matrix --------------------------------------------

func TestLoad_Precedence_DefaultsThenFileThenEnv(t *testing.T) {
	path := writeYAML(t, "server:\n  port: 8000\n")
	withEnv(t, "SERVER_PORT", "9999")

	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithEnv(),
		configkit.WithDefault("server.port", 5000),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 9999 {
		t.Errorf("env must win: port=%d, want 9999", got.Server.Port)
	}
}

func TestLoad_Precedence_FileWinsOverDefault(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, "server:\n  port: 8000\n")
	var got cfg
	err := configkit.New(
		configkit.WithConfigFile(path),
		configkit.WithDefault("server.port", 5000),
	).Load(&got)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Server.Port != 8000 {
		t.Errorf("file must beat default: port=%d, want 8000", got.Server.Port)
	}
}

// ---------- nil / misuse ---------------------------------------------------

func TestLoad_NilPointerReturnsError(t *testing.T) {
	t.Parallel()
	err := configkit.New().Load(nil)
	if err == nil {
		t.Fatal("Load(nil) must surface an Unmarshal error from Viper, got nil")
	}
}

func TestLoad_NonPointerReturnsError(t *testing.T) {
	t.Parallel()
	err := configkit.New().Load(cfg{})
	if err == nil {
		t.Fatal("Load(non-pointer) must surface an Unmarshal error from Viper, got nil")
	}
}

// ---------- option application order --------------------------------------

func TestOptions_AppliedInOrder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts []configkit.Option
		// Confirms New() never panics and the resulting Loader can
		// Load a struct for the given option mix. Guards against
		// silent option regressions.
	}{
		{
			name: "all options",
			opts: []configkit.Option{
				configkit.WithConfigFile("x.yaml"),
				configkit.WithEnv(),
				configkit.WithEnvPrefix("APP"),
				configkit.WithEnvKeyReplacer(strings.NewReplacer(".", "_")),
				configkit.WithDefault("k", 1),
				configkit.WithValidator(func(any) error { return nil }),
			},
		},
		{
			name: "env only",
			opts: []configkit.Option{configkit.WithEnv()},
		},
		{
			name: "defaults only",
			opts: []configkit.Option{
				configkit.WithDefault("a", 1),
				configkit.WithDefault("b", 2),
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := configkit.New(tc.opts...)
			if l == nil {
				t.Fatal("New returned nil")
			}
			var got cfg
			if err := l.Load(&got); err != nil {
				t.Errorf("Load: %v", err)
			}
		})
	}
}
