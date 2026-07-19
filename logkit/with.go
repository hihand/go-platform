package logkit

// With appends the supplied attrs to the logger and returns a new
// logger that includes them on every subsequent call. The original
// logger is untouched. This is the canonical way to derive a scoped
// logger (e.g. a per-request logger carrying request.id).
//
// Precedence (highest wins): call attrs > With attrs > context attrs >
// static resource fields. With attrs override parent logger With attrs
// by appending — duplicates favour the later attribute.
func (l *impl) With(attrs ...Attr) Logger {
	if len(attrs) == 0 {
		return l
	}
	cp := *l
	merged := make([]Attr, 0, len(l.withAttrs)+len(attrs))
	merged = append(merged, l.withAttrs...)
	merged = append(merged, attrs...)
	cp.withAttrs = merged
	return &cp
}
