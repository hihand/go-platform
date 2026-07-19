package paginationkit

// NewCursorPaginationRequest builds a CursorRequest with normalised
// Limit and untouched cursor tokens. Paginationkit treats After and
// Before as opaque strings on purpose — it does not parse, validate,
// or encode them — so they are stored verbatim on the returned
// request. Pass "" for either side to leave it unbound ("start from
// the beginning", "no upper bound").
//
// Examples:
//
//	NewCursorPaginationRequest("", "", 0)   // first page, default limit
//	NewCursorPaginationRequest("eyJpZCI6MTB9", "", 50) // forward from a cursor
//	NewCursorPaginationRequest("", "eyJpZCI6NTB9", 25) // backward from a cursor
func NewCursorPaginationRequest(after, before string, limit int) CursorRequest {
	return CursorRequest{
		BaseRequest: BaseRequest{Limit: normaliseLimit(limit)},
		After:       after,
		Before:      before,
	}
}

// NewCursorPaginationResponse builds a CursorResponse from the raw
// values a service layer computed for a page. nextCursor / prevCursor
// may be nil to signal "no next/previous page" (which is also implied
// by hasNext == false / hasPrevious == false, but the cursor strings
// are the canonical source of truth for cursor pagination).
//
// The Limit is normalised at construction time so the value the
// caller reads on the response matches the value the caller used on
// the request.
func NewCursorPaginationResponse(limit int, nextCursor, prevCursor *string, hasNext, hasPrevious bool) CursorResponse {
	return CursorResponse{
		BaseResponse: BaseResponse{
			Limit:       normaliseLimit(limit),
			HasNext:     hasNext,
			HasPrevious: hasPrevious,
		},
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
	}
}