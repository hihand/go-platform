package logkit

import "time"

// Record is the internal value passed from the hot path to the encoder.
//
// It exists for one reason: encoder.emit previously took nine positional
// parameters (writer, buffer, time, level, message, event, static, merged,
// caller). The parameter list was hard to read and easy to mis-order at
// the call site. Record collapses them into a single value the encoder
// reads from. It is intentionally unexported — callers never construct
// or hold a Record.
//
// Record is not an abstraction layer. It does not implement an interface,
// it carries no behaviour, and it is consumed exactly once per log call.
// It is a plain data carrier.
type Record struct {
	ts     time.Time
	level  Level
	msg    string
	event  string
	caller string
	static []Attr // service.* / deployment.* — set once per Logger
	attrs  []Attr // merged: withAttrs → ctxAttrs → callAttrs, last write wins
}