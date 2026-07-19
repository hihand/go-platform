package logkit

// Level is the typed enum for the four log severities. Using a typed
// value (instead of a string) prevents typos at the call site and lets
// the compiler enforce exhaustiveness.
//
// Levels are ordered: DEBUG < INFO < WARN < ERROR. The logger drops
// records whose level is below WithMinLevel; passing any unknown value
// to WithMinLevel is clamped to INFO so configuration never panics.
//
// The canonical wire label is produced by Level.String — the same
// string is used internally by the encoder, so any third-party
// decoder can rely on it.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the canonical, uppercase level label used on the
// wire. Unknown values render as "INFO" so a misconfigured logger
// still emits readable output rather than silently dropping records.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// clampLevel clamps an arbitrary Level to the canonical range. The
// fallback for an unrecognised value is LevelInfo — the logger never
// panics on configuration.
func clampLevel(l Level) Level {
	if l < LevelDebug || l > LevelError {
		return LevelInfo
	}
	return l
}

// Key is the typed enum for the canonical attribute/field names that
// appear on every log record. The constants below cover the schema
// fields the encoder always emits (timestamp, level, message, event,
// caller) plus the common static resource fields (service.*,
// deployment.environment). For application-defined fields (e.g.
// "payment.id", "user.email"), use the AnyKey escape hatch.
type Key string

const (
	// Schema fields — emitted by the encoder.
	KeyEvent     Key = "event"
	KeyTimestamp Key = "timestamp"
	KeyLevel     Key = "level"
	KeyMessage   Key = "message"
	KeyCaller    Key = "caller"

	// Static resource fields — set via WithService / WithDeployment.
	KeyServiceName           Key = "service.name"
	KeyServiceVersion        Key = "service.version"
	KeyDeploymentEnvironment Key = "deployment.environment"
)

// AnyKey constructs a Key from an arbitrary string. Use the typed
// constants directly when one applies — callers reading the code
// should immediately see known field names without having to grep.
func AnyKey(name string) Key { return Key(name) }

// String makes Key satisfy fmt.Stringer so values render naturally in
// test diagnostics and errors.
func (k Key) String() string { return string(k) }