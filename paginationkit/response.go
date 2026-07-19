package paginationkit

// BaseResponse is the common shape every pagination response in
// this package composes. It carries the always-known information
// (the effective Limit, and the booleans describing whether more
// pages exist in each direction). Optional fields — Total, and the
// cursor tokens — live on the concrete response types so that
// "absent" can be distinguished from "zero".
//
// Offset and Cursor response types embed BaseResponse rather than
// duplicating those fields, mirroring the request side.
type BaseResponse struct {
	// Limit is the effective page size used to produce this
	// response. After normalisation, this is always in
	// [1, MaxPaginationLimit].
	Limit int `json:"limit"`

	// HasNext is true when at least one more item exists after
	// this page. False means "this was the last page" (the
	// caller should stop paging forward).
	HasNext bool `json:"has_next"`

	// HasPrevious is true when at least one item exists before
	// this page. False means "this is the first page" (the
	// caller should stop paging backward).
	HasPrevious bool `json:"has_previous"`
}

// OffsetResponse describes the page returned for an OffsetRequest.
// In addition to the always-known fields inherited from
// BaseResponse, it carries Total (the total number of items
// matching the underlying query) as an optional pointer so that
// callers that do not want to pay for a COUNT(*) can simply leave
// it nil.
type OffsetResponse struct {
	BaseResponse

	// Total is the total number of items in the underlying
	// collection that match the query — i.e. the count of items
	// that *would* be returned by paging through every page. nil
	// means "unknown / not computed". A pointer is used (instead
	// of a plain int) so callers and serialisers can tell the
	// difference between "0 items" and "we did not compute the
	// total".
	Total *int64 `json:"total,omitempty"`
}

// CursorResponse describes the page returned for a CursorRequest.
// In addition to the always-known fields inherited from
// BaseResponse, it carries the NextCursor and PrevCursor tokens
// the caller should use for forward and backward paging.
//
// Both cursor fields are strings (cursors are opaque to this
// package) and pointers (so the absence of a cursor is
// distinguishable from an empty-string cursor).
type CursorResponse struct {
	BaseResponse

	// NextCursor is the cursor the caller should pass as
	// CursorRequest.After to fetch the page that follows this
	// one. nil (or an empty string after dereference) means
	// "there is no next page" — equivalent to HasNext == false.
	NextCursor *string `json:"next_cursor,omitempty"`

	// PrevCursor is the cursor the caller should pass as
	// CursorRequest.Before to fetch the page that precedes this
	// one. nil (or an empty string after dereference) means
	// "there is no previous page" — equivalent to HasPrevious
	// == false.
	PrevCursor *string `json:"prev_cursor,omitempty"`
}