package paginationkit

// BaseRequest is the common shape every pagination request in this
// package composes. It is intentionally minimal: only the Limit
// field, because both offset and cursor pagination need to know how
// many items to fetch per page.
//
// Offset and Cursor request types embed (rather than duplicate)
// BaseRequest, so the Limit handling and validation live in exactly
// one place. Adding a new pagination style in the future — for
// example, "page number" — only requires embedding BaseRequest and
// adding the new style-specific fields.
type BaseRequest struct {
	// Limit is the maximum number of items the caller wants in a
	// single page. It is normalised at construction time to a value
	// in [1, MaxPaginationLimit]; the constructor never stores a
	// raw, caller-supplied value.
	Limit int `json:"limit"`
}

// OffsetRequest describes a classic offset+limit pagination query.
// "Skip the first Offset items, then return up to Limit items."
//
// Use this when the caller can tolerate the cost of an offset scan
// (typically: back-office UIs, small datasets, or pages deep enough
// that the cost is acceptable). For large or unbounded streams,
// prefer CursorRequest.
type OffsetRequest struct {
	BaseRequest
	// Offset is the zero-based index of the first item to return.
	// Negative values are clamped to 0 by the constructor.
	Offset int `json:"offset"`
}

// CursorRequest describes a keyset (cursor-based) pagination query.
// "Return up to Limit items that come after the After cursor, or
// before the Before cursor."
//
// After and Before are opaque strings whose format is decided by the
// producer (an encoded offset, a base64 id, a timestamp, …). The
// paginationkit package treats them as opaque on purpose: it does
// not parse, validate, or encode them.
//
// Cursor pagination is preferable to offset when:
//   - the dataset is large or unbounded;
//   - rows are inserted/deleted while paging (offset would skip or
//     repeat items);
//   - latency must stay constant regardless of how deep the caller
//     has paged.
type CursorRequest struct {
	BaseRequest
	// After is the cursor that anchors the start of the page. The
	// producer decides its meaning (typically "items strictly after
	// this key"). An empty string means "start from the beginning".
	After string `json:"after,omitempty"`
	// Before is the cursor that anchors the end of the page. The
	// producer decides its meaning (typically "items strictly
	// before this key"). An empty string means "no upper bound".
	Before string `json:"before,omitempty"`
}