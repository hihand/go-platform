package logkit

import (
	"io"
	"math"
	"strconv"
	"time"
)

// encoder writes one JSON object line per call. The implementation is
// manual (no encoding/json) because the hot path needs to amortise
// allocations across calls and avoid intermediate interface boxing.
//
// Layout per record:
//
//	{"timestamp":"...","level":"...","message":"...","event":"...","key1":v1,...}\n
//
// All Attr visitors write directly into a sync.Pool-backed scratch
// buffer, then a single Write keeps the I/O path tight.
type encoder struct{}

// defaultEncoder is a singleton shared across Logger instances — the
// type is stateless.
func defaultEncoder() encoder { return encoder{} }

// emit writes a complete log record to w. The caller owns buffer reuse:
// pass the same buf in tight loops to keep allocations flat.
//
// r is the already-built Record (see record.go). The hot path in log.go
// owns the merge so precedence rules are explicit at the call site;
// the encoder only walks r.attrs in order.
func (encoder) emit(w io.Writer, buf []byte, r Record) ([]byte, error) {
	buf = buf[:0]
	buf = append(buf, '{')

	// Timestamp: write straight into buf. time.Format would allocate
	// a ~32 B string on every call, defeating the pool.
	buf = append(buf, `"`+string(KeyTimestamp)+`":"`...)
	buf = r.ts.UTC().AppendFormat(buf, time.RFC3339Nano)
	buf = append(buf, `",`...)

	buf = appendKeyStringValue(buf, KeyLevel, r.level.String())
	buf = appendKeyStringValue(buf, KeyMessage, r.msg)
	if r.event != "" {
		buf = appendKeyStringValue(buf, KeyEvent, r.event)
	}

	for i := range r.static {
		buf = appendAttr(buf, r.static[i])
	}
	for i := range r.attrs {
		buf = appendAttr(buf, r.attrs[i])
	}

	if r.caller != "" {
		buf = appendKeyStringValue(buf, KeyCaller, r.caller)
	}

	// strip trailing comma to keep the JSON valid
	if n := len(buf); n > 1 && buf[n-1] == ',' {
		buf = buf[:n-1]
	}
	buf = append(buf, "}\n"...)

	if _, err := w.Write(buf); err != nil {
		return buf, err
	}
	return buf, nil
}

// appendKeyStringValue writes "key":"escaped-value", preserving the
// trailing comma for the next field. The key is taken as a Key so
// canonical schema names stay typed.
func appendKeyStringValue(buf []byte, key Key, value string) []byte {
	buf = append(buf, '"')
	buf = append(buf, key...)
	buf = append(buf, `":"`...)
	buf = appendJSONString(buf, value)
	buf = append(buf, `",`...)
	return buf
}

// appendAttr dispatches on Attr.kind and writes the rendered key/value
// pair with a trailing comma.
func appendAttr(buf []byte, a Attr) []byte {
	switch a.kind {
	case attrString:
		return appendKeyStringValue(buf, a.key, a.str)
	case attrInt:
		buf = append(buf, '"')
		buf = append(buf, string(a.key)...)
		buf = append(buf, `":`...)
		buf = strconv.AppendInt(buf, int64(a.num), 10)
	case attrInt64:
		buf = append(buf, '"')
		buf = append(buf, string(a.key)...)
		buf = append(buf, `":`...)
		buf = strconv.AppendInt(buf, int64(a.num), 10)
	case attrUint64:
		buf = append(buf, '"')
		buf = append(buf, string(a.key)...)
		buf = append(buf, `":`...)
		buf = strconv.AppendUint(buf, a.num, 10)
	case attrFloat64:
		buf = append(buf, '"')
		buf = append(buf, string(a.key)...)
		buf = append(buf, `":`...)
		buf = strconv.AppendFloat(buf, math.Float64frombits(a.num), 'g', -1, 64)
	case attrBool:
		buf = append(buf, '"')
		buf = append(buf, string(a.key)...)
		buf = append(buf, `":`...)
		buf = strconv.AppendBool(buf, a.bool)
	case attrDur:
		buf = append(buf, '"')
		buf = append(buf, string(a.key)...)
		buf = append(buf, `":`...)
		buf = strconv.AppendInt(buf, int64(a.num), 10)
	case attrTime:
		buf = append(buf, '"')
		buf = append(buf, string(a.key)...)
		buf = append(buf, `":"`...)
		buf = a.time.UTC().AppendFormat(buf, time.RFC3339Nano)
		buf = append(buf, '"')
	}
	buf = append(buf, ',')
	return buf
}

// appendJSONString escapes a string for inclusion inside a JSON double-
// quoted value. It only escapes the characters JSON requires; it does
// not validate UTF-8 (the caller is expected to own input).
func appendJSONString(buf []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\':
			buf = append(buf, '\\', c)
		case '\n':
			buf = append(buf, '\\', 'n')
		case '\r':
			buf = append(buf, '\\', 'r')
		case '\t':
			buf = append(buf, '\\', 't')
		default:
			if c < 0x20 {
				buf = append(buf, '\\', 'u', '0', '0', '0', '0'+c>>4)
				if c&0xf < 10 {
					buf = append(buf, '0'+c&0xf)
				} else {
					buf = append(buf, 'a'+c&0xf-10)
				}
				continue
			}
			buf = append(buf, c)
		}
	}
	return buf
}