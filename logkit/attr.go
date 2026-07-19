package logkit

import (
	"math"
	"time"
)

// Attr is a single typed log attribute. It mirrors slog.Attr in shape
// but uses logkit-owned types so the package can evolve independently.
//
// Use the typed constructors (String, Int, Int64, Uint64, Float64, Bool,
// Dur, Time) with the canonical Key constants from spec.go or
// AnyKey for user-defined names. The package keeps internal encoding
// allocation-free on the hot path by visiting Attr directly.
type Attr struct {
	key  Key
	kind attrKind
	str  string
	num  uint64 // int64/uint64 or float64 bits
	bool bool
	time time.Time
}

type attrKind uint8

const (
	attrString attrKind = iota
	attrInt
	attrInt64
	attrUint64
	attrFloat64
	attrBool
	attrDur
	attrTime
)

// String creates a string attribute. Hot path: most business fields are
// strings — keep this constructor cheap.
func String(key Key, value string) Attr {
	return Attr{key: key, kind: attrString, str: value}
}

// Int creates an int attribute. Stored as int64 internally to keep the
// hot-path emitter branch-free.
func Int(key Key, value int) Attr {
	return Attr{key: key, kind: attrInt, num: uint64(int64(value))}
}

// Int64 creates an int64 attribute.
func Int64(key Key, value int64) Attr {
	return Attr{key: key, kind: attrInt64, num: uint64(value)}
}

// Uint64 creates a uint64 attribute.
func Uint64(key Key, value uint64) Attr {
	return Attr{key: key, kind: attrUint64, num: value}
}

// Float64 creates a float64 attribute. The IEEE-754 bits are stored
// so the emitter can hand them straight to strconv.AppendFloat
// without re-bitshifting.
func Float64(key Key, value float64) Attr {
	return Attr{key: key, kind: attrFloat64, num: math.Float64bits(value)}
}

// Bool creates a bool attribute.
func Bool(key Key, value bool) Attr {
	return Attr{key: key, kind: attrBool, bool: value}
}

// Dur creates a duration attribute. Stored as int64 nanoseconds.
func Dur(key Key, value time.Duration) Attr {
	return Attr{key: key, kind: attrDur, num: uint64(int64(value))}
}

// Time creates a time attribute. Stored as time.Time so the encoder can
// format directly via time.AppendFormat on the hot path.
func Time(key Key, value time.Time) Attr {
	return Attr{key: key, kind: attrTime, time: value}
}