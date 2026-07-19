package paginationkit

// DefaultPaginationLimit is the fallback value used by the constructors
// when a caller asks for a non-positive Limit. It is intentionally
// small: most APIs that wire pagination into their transport return
// pages of 10–50 items.
const DefaultPaginationLimit int = 20

// MaxPaginationLimit is the hard ceiling enforced by the constructors.
// Any larger Limit is silently clamped down. It exists to protect
// downstream layers (databases, renderers) from accidental
// "give me everything" requests, regardless of how the request
// arrived (HTTP query, gRPC message, internal call).
const MaxPaginationLimit int = 100

// normaliseLimit applies the Default / Max clamp to a raw Limit value
// coming from a request. It is shared by both offset and cursor code
// paths because the rule is identical: a page must be at least 1 and
// must not exceed MaxPaginationLimit.
//
// The function tolerates any int input (including negative numbers)
// without panicking, so callers can hand it raw transport values
// directly.
func normaliseLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultPaginationLimit
	case limit > MaxPaginationLimit:
		return MaxPaginationLimit
	default:
		return limit
	}
}

// normaliseOffset clamps a raw offset to a non-negative value. Negative
// offsets are treated as "start from the beginning". Used by the
// offset request path only.
func normaliseOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}