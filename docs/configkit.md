# configkit

A thin, idiomatic wrapper around [Viper](https://github.com/spf13/viper) that loads configuration from a YAML file and environment variables, then unmarshals the result into a caller-defined struct.

## At a glance

```go
loader := configkit.New(
    configkit.WithConfigFile("config.yaml"),
    configkit.WithEnv(),
    configkit.WithEnvPrefix("APP"),
    configkit.WithDefaults(map[string]any{
        "server.host": "0.0.0.0",
        "server.port": 8080,
    }),
    configkit.WithValidator(func(c any) error {
        // plug a struct-validator library here
        return nil
    }),
)

var cfg AppConfig
if err := loader.Load(&cfg); err != nil {
    log.Fatal(err)
}

// cfg.Server.Host / cfg.Server.Port are now populated, with env
// overriding file overriding defaults.
```

## Design contract

- **Two-call surface.** `configkit.New(opts...)` returns a `Loader`; `loader.Load(cfg)` fills a caller-defined struct. Everything else is an `Option`.
- **Caller owns the shape.** `AppConfig`, `DatabaseConfig`, `JWTConfig` — all application types. `configkit` does not import any of them, does not define them, and does not look at them with reflection.
- **Viper is a private detail.** No public type, interface, or signature mentions `*viper.Viper`. Swapping the underlying library is a one-file change inside the package.
- **Loader is re-runnable.** `Load()` builds a fresh Viper instance on every call. Callers can re-load after editing the config file or rotating env vars without rebuilding the Loader, and the same Loader is safe to share across goroutines.
- **A missing config file is not an error.** `viper.ConfigFileNotFoundError` and `os.IsNotExist` are both swallowed so an application can boot from defaults + env vars alone.
- **No global state.** No `init()`, no package-level mutability, no hidden registries.
- **Validates, but does not define a validator.** `WithValidator` accepts any `func(any) error` and runs it verbatim after a successful Unmarshal.

## Loading precedence

Inside `Load()`, last source wins:

```
defaults  <  config file  <  environment
```

The `Loader` itself does not expose which source supplied a value. If you need to know, run `Load()` twice — once with `WithEnv()` to capture the "with env" baseline, once without to capture the "without env" baseline — and compare.

A typical YAML file:

```yaml
server:
  host: 0.0.0.0
  port: 8080
database:
  url: postgres://localhost/app
  max_conns: 5
log:
  level: info
```

Bound env vars (when `WithEnv()` + `WithEnvPrefix("APP")` are wired):

```bash
APP_SERVER_HOST=api.example.com   # → server.host
APP_SERVER_PORT=9090              # → server.port
APP_LOG_LEVEL=warn                # → log.level
```

The default key replacer (`strings.NewReplacer(".", "_")`) maps `server.port` ↔ `SERVER_PORT`, so the env var name matches the dotted Viper key with dots replaced by underscores, prefixed by `APP_`.

## Public API

### `Loader`

```go
type Loader interface {
    Load(cfg any) error
}
```

A `Loader` is built once with `New` and can be used as many times as needed. Each `Load` call is independent: it builds a fresh Viper instance, re-reads the file from disk, re-reads the env from the process environment, applies the registered defaults, then unmarshals into `cfg`.

### `New`

```go
func New(opts ...Option) Loader
```

A zero-option `New()` returns a `Loader` that:

- has no config file,
- does not read env vars,
- has no defaults,
- has no validator.

Such a `Loader` still works: `Load()` unmarshals an empty Viper instance into `cfg` and returns `nil`. Callers who need file / env / defaults / validation wire them in via `With*` options.

### `Option`

`Option` is a functional option:

```go
type Option func(*impl)
```

Options are applied in the order supplied. Later options override earlier ones for scalar fields (`prefix`, `replacer`, `validator`, `file path`). Map-shaped options (`WithDefault`, `WithDefaults`) merge — there is no concept of "replace the defaults map".

## Options

### `WithConfigFile`

```go
configkit.WithConfigFile("config.yaml")
```

Sets the path to the YAML configuration file. The file is read by `Load()` via Viper's `ReadInConfig`. Either flavor of "missing" error — `viper.ConfigFileNotFoundError` from the discovery path, or `os.IsNotExist` from an explicit `SetConfigFile` path — is swallowed so an application can boot from env vars alone.

An empty path is a no-op so callers can forward user-supplied paths without a guard.

### `WithEnv`

```go
configkit.WithEnv()
```

Enables binding of every environment variable into Viper. Without this option, env vars are ignored regardless of any prefix or key replacer that may be configured.

Call this once before `WithEnvPrefix` / `WithEnvKeyReplacer` — the order of options only matters when the same option is repeated.

### `WithEnvPrefix`

```go
configkit.WithEnvPrefix("APP")
```

Sets the prefix that env vars must carry to be visible to Viper. Empty prefix disables the prefix.

```go
WithEnvPrefix("APP"), env var APP_SERVER_PORT → key "server.port"
```

The prefix is applied by Viper after `WithEnvKeyReplacer`; both are independent knobs.

### `WithEnvKeyReplacer`

```go
configkit.WithEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
```

Sets the replacer used to normalise env-var names into Viper keys. The default is `strings.NewReplacer(".", "_")` so `server.port` maps to env var `SERVER_PORT`.

Pass `nil` to keep the default; pass your own `*strings.Replacer` to customise (e.g. also accept dashed names).

### `WithDefault`

```go
configkit.WithDefault("server.port", 8080)
```

Registers a single fallback value for the given key. Multiple calls register multiple defaults; precedence inside `Load()` is `defaults < config file < environment`, so a default is masked whenever the file or the environment supplies a value for the same key.

`key` is a dotted Viper path (e.g. `"server.port"`). `value` is any value that Viper's `Set` accepts. Empty keys are skipped silently.

### `WithDefaults`

```go
configkit.WithDefaults(map[string]any{
    "server.host": "0.0.0.0",
    "server.port": 8080,
})
```

Bulk variant of `WithDefault`. Each entry follows the same precedence rules and the same dotted key convention. A `nil` or empty map is a no-op; empty keys inside the map are skipped silently so callers can forward user-supplied maps without a guard.

Use `WithDefault` when adding one key at a time inside a conditional; use `WithDefaults` when the defaults are known up front (e.g. compile-time constants, a defaults struct).

### `WithValidator`

```go
configkit.WithValidator(func(c any) error {
    return validate.Struct(c) // e.g. go-playground/validator
})
```

Installs a post-unmarshal validator. The validator runs only after Viper's `Unmarshal` has succeeded and receives the same value the caller passed to `Load()`.

Pass `nil` to clear any previously installed validator. A validator that returns a non-nil error is returned to the caller verbatim — no wrapping, no translation.

## Behaviour reference

| Scenario                                                              | Behaviour                                            |
|-----------------------------------------------------------------------|------------------------------------------------------|
| `New()` with no options                                               | Returns a usable `Loader` that fills a zero struct.   |
| `WithConfigFile("")`                                                 | No-op.                                               |
| `WithConfigFile("absent.yaml")`                                       | Silently ignored — `Load()` returns `nil`.           |
| `WithConfigFile("malformed.yaml")`                                    | Viper's parse error returned verbatim.               |
| `WithEnv()` only (no prefix, no replacer change)                      | Every env var visible to Viper; `SERVER_PORT` → `server.port`. |
| `WithEnv()` + `WithEnvPrefix("APP")`                                  | Only `APP_*` env vars; `APP_SERVER_PORT` → `server.port`. |
| `WithEnvKeyReplacer(nil)`                                             | Keeps the default `.` → `_` replacer.                |
| `WithDefault("")` or empty key inside `WithDefaults`                  | Silently skipped.                                    |
| `WithDefaults(nil)`                                                  | No-op.                                               |
| `WithValidator(nil)`                                                  | Clears a previously installed validator.             |
| `WithValidator` returns error                                         | `Load()` returns that error verbatim, no wrap.       |
| Same `Loader` shared across goroutines, each calling `Load`           | Safe; each call rebuilds Viper from scratch.         |
| `Load(&cfg)` then `Load(&cfg)` again                                  | Both calls succeed; file and env are re-read each time. |

## Patterns

### Boot from env alone

The missing-file behaviour makes local development easy: ship a `.env` file with overrides, omit `config.yaml`, and the app still boots with sane defaults.

```go
loader := configkit.New(
    configkit.WithDefaults(map[string]any{
        "server.host": "localhost",
        "server.port": 3000,
    }),
    configkit.WithEnv(),
    configkit.WithEnvPrefix("APP"),
)
```

### Validator with `go-playground/validator`

```go
import "github.com/go-playground/validator/v10"

var validate = validator.New()

loader := configkit.New(
    configkit.WithConfigFile("config.yaml"),
    configkit.WithValidator(func(c any) error {
        return validate.Struct(c)
    }),
)
```

### Per-environment overrides without rebuilding the Loader

```go
loader := configkit.New(
    configkit.WithConfigFile("config.yaml"),
    configkit.WithDefaults(map[string]any{
        "log.level": "info",
    }),
)

// Staging
os.Setenv("APP_LOG_LEVEL", "debug")
var stagingCfg Config
_ = loader.Load(&stagingCfg)

// Production
os.Setenv("APP_LOG_LEVEL", "warn")
var prodCfg Config
_ = loader.Load(&prodCfg)
```

### Forwarding user-supplied paths

Empty paths and empty maps are intentional no-ops so callers can pass user input straight through:

```go
opts := []configkit.Option{
    configkit.WithConfigFile(userPath),         // empty → no-op
    configkit.WithEnv(),
    configkit.WithDefaults(userDefaults),       // nil → no-op
}
loader := configkit.New(opts...)
```

## Anti-patterns

- **Don't call `WithConfigFile` more than once.** The second call overwrites the first — use one canonical config file path. For per-environment files, choose at the call site (`if env == "prod" { WithConfigFile("prod.yaml") } else { WithConfigFile("dev.yaml") }`).
- **Don't put secrets in defaults.** Defaults are visible in source — keep secrets in env vars or a secret manager.
- **Don't rely on the Loader for cross-cutting concerns.** configkit is config-only. Logging, metrics, and request tracing live in `logkit`, `contextkit`, and OTel respectively.
- **Don't expect partial unmarshalling.** Viper's `Unmarshal` is all-or-nothing — a malformed YAML is a single error, not "some fields filled, some missing".

## File Layout

| File                  | Purpose                                                                                |
|-----------------------|----------------------------------------------------------------------------------------|
| `spec.go`             | Package doc + `Loader` interface                                                        |
| `new.go`              | `impl` struct + `New(opts ...Option) Loader`                                             |
| `option.go`           | Functional options: `WithConfigFile`, `WithEnv`, `WithEnvPrefix`, `WithEnvKeyReplacer`, `WithDefault`, `WithDefaults`, `WithValidator` |
| `load.go`             | `Load` implementation + `loadFile` + `applyDefaults` helpers                           |
| `common.go`           | `configFileType` constant (shared between option and loader)                            |
| `configkit_test.go`   | All package tests, table-driven, parallel, race-clean                                   |
| `example_test.go`     | Runnable godoc examples                                                                 |
