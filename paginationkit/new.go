package paginationkit

// This file holds sugar / one-liner constructors. The "primary"
// constructors (NewOffsetPaginationRequest, NewOffsetPaginationResponse,
// NewCursorPaginationRequest, NewCursorPaginationResponse) already
// live in offset.go and cursor.go, one concern per file, per
// RULES.md. The functions below are sugar wrappers around them for
// the most common call sites, so handlers can stay compact.
//
// Sugar constructors:
//
//	FirstPage(limit)               — offset request starting at 0
//	EmptyOffsetResponse(limit)     — zero-page offset response
//	EmptyCursorResponse(limit)     — zero-page cursor response
//
// Sugar is intentionally minimal: the public surface stays tiny and
// predictable, and everything still flows through the normalising
// constructors defined in offset.go / cursor.go.

// FirstPage returns an OffsetRequest for the first page of results.
// limit <= 0 falls back to DefaultPaginationLimit; limit above
// MaxPaginationLimit is clamped. Equivalent to:
//
//	NewOffsetPaginationRequest(0, limit)
//
// but reads more naturally at the call site.
func FirstPage(limit int) OffsetRequest {
	return NewOffsetPaginationRequest(0, limit)
}

// EmptyOffsetResponse returns an OffsetResponse that represents an
// empty page: no items, no next, no previous, no Total. It is the
// canonical "I queried, there was nothing" response for the offset
// path. limit is normalised like every other constructor.
func EmptyOffsetResponse(limit int) OffsetResponse {
	return NewOffsetPaginationResponse(limit, nil, false, false)
}

// EmptyCursorResponse returns a CursorResponse that represents an
// empty page: no items, no next, no previous, no cursor tokens. It
// is the canonical "I queried, there was nothing" response for the
// cursor path. limit is normalised like every other constructor.
func EmptyCursorResponse(limit int) CursorResponse {
	return NewCursorPaginationResponse(limit, nil, nil, false, false)
}