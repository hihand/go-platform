package paginationkit

// NewOffsetPaginationRequest builds an OffsetRequest with normalised
// values. Both arguments are tolerated as raw input from a transport
// layer (HTTP query, gRPC message, JSON body, internal call): the
// returned request will have a Limit in [1, MaxPaginationLimit] and a
// non-negative Offset.
//
// Examples:
//
//	NewOffsetPaginationRequest(0, 0)   // first page, default limit
//	NewOffsetPaginationRequest(50, 10) // second page of 50
//	NewOffsetPaginationRequest(-5, 999) // offset clamped to 0, limit clamped to MaxPaginationLimit
func NewOffsetPaginationRequest(offset, limit int) OffsetRequest {
	return OffsetRequest{
		BaseRequest: BaseRequest{Limit: normaliseLimit(limit)},
		Offset:      normaliseOffset(offset),
	}
}

// NewOffsetPaginationResponse builds an OffsetResponse from the raw
// values a service layer computed for a page. The four booleans and
// the total count are taken verbatim; this constructor does not try
// to derive HasNext/HasPrevious from Total (it cannot, in general:
// a service layer may already know the answer without having run
// COUNT(*)).
//
// total may be nil when the service chose not to compute it. Passing
// nil keeps Total as nil on the returned response, so the wire
// representation stays "absent" rather than "0".
func NewOffsetPaginationResponse(limit int, total *int64, hasNext, hasPrevious bool) OffsetResponse {
	return OffsetResponse{
		BaseResponse: BaseResponse{
			Limit:       normaliseLimit(limit),
			HasNext:     hasNext,
			HasPrevious: hasPrevious,
		},
		Total: total,
	}
}