package paginationkit_test

import (
	"encoding/json"
	"testing"

	"github.com/hihand/go-platform/paginationkit"
)

// ---------- Constants ----------------------------------------------------

func TestConstants_HaveSensibleDefaults(t *testing.T) {
	t.Parallel()
	if paginationkit.DefaultPaginationLimit <= 0 {
		t.Errorf("DefaultPaginationLimit must be > 0, got %d", paginationkit.DefaultPaginationLimit)
	}
	if paginationkit.MaxPaginationLimit <= paginationkit.DefaultPaginationLimit {
		t.Errorf("MaxPaginationLimit (%d) must be greater than DefaultPaginationLimit (%d)",
			paginationkit.MaxPaginationLimit, paginationkit.DefaultPaginationLimit)
	}
}

// ---------- Offset request -----------------------------------------------

func TestNewOffsetPaginationRequest_NormalisationMatrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		offset     int
		limit      int
		wantOffset int
		wantLimit  int
	}{
		{"zero_zero", 0, 0, 0, paginationkit.DefaultPaginationLimit},
		{"positive", 50, 10, 50, 10},
		{"negative_offset_clamps_to_zero", -5, 25, 0, 25},
		{"negative_limit_uses_default", 10, -1, 10, paginationkit.DefaultPaginationLimit},
		{"zero_limit_uses_default", 10, 0, 10, paginationkit.DefaultPaginationLimit},
		{"limit_above_max_clamped", 10, 9999, 10, paginationkit.MaxPaginationLimit},
		{"limit_at_max_kept", 0, paginationkit.MaxPaginationLimit, 0, paginationkit.MaxPaginationLimit},
		{"limit_one_kept", 0, 1, 0, 1},
		{"both_negative", -100, -100, 0, paginationkit.DefaultPaginationLimit},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := paginationkit.NewOffsetPaginationRequest(tc.offset, tc.limit)
			if req.Offset != tc.wantOffset {
				t.Errorf("offset: want %d, got %d", tc.wantOffset, req.Offset)
			}
			if req.Limit != tc.wantLimit {
				t.Errorf("limit: want %d, got %d", tc.wantLimit, req.Limit)
			}
		})
	}
}

func TestOffsetRequest_ComposesBaseRequest(t *testing.T) {
	t.Parallel()
	req := paginationkit.NewOffsetPaginationRequest(20, 50)
	// Limit is reachable through the embedded BaseRequest.
	if req.BaseRequest.Limit != 50 {
		t.Errorf("embedded BaseRequest.Limit: want 50, got %d", req.BaseRequest.Limit)
	}
	if req.Limit != 50 {
		t.Errorf("promoted Limit: want 50, got %d", req.Limit)
	}
}

func TestOffsetRequest_JSONShape(t *testing.T) {
	t.Parallel()
	req := paginationkit.NewOffsetPaginationRequest(40, 25)
	buf, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Round-trip and verify the fields survived.
	var got paginationkit.OffsetRequest
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Offset != 40 || got.Limit != 25 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

// ---------- Cursor request -----------------------------------------------

func TestNewCursorPaginationRequest_NormalisationMatrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		after     string
		before    string
		limit     int
		wantLimit int
	}{
		{"empty_empty", "", "", 0, paginationkit.DefaultPaginationLimit},
		{"forward", "cursor-a", "", 25, 25},
		{"backward", "", "cursor-b", 25, 25},
		{"both", "cursor-a", "cursor-b", 50, 50},
		{"negative_limit", "x", "y", -10, paginationkit.DefaultPaginationLimit},
		{"limit_clamped", "x", "y", 9999, paginationkit.MaxPaginationLimit},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := paginationkit.NewCursorPaginationRequest(tc.after, tc.before, tc.limit)
			if req.After != tc.after {
				t.Errorf("after: want %q, got %q", tc.after, req.After)
			}
			if req.Before != tc.before {
				t.Errorf("before: want %q, got %q", tc.before, req.Before)
			}
			if req.Limit != tc.wantLimit {
				t.Errorf("limit: want %d, got %d", tc.wantLimit, req.Limit)
			}
		})
	}
}

func TestCursorRequest_CursorTokensAreOpaque(t *testing.T) {
	t.Parallel()
	// Paginationkit must not interpret cursor strings. Pass raw,
	// recognisable tokens and check they are stored verbatim.
	req := paginationkit.NewCursorPaginationRequest("eyJpZCI6MTB9", "eyJpZCI6MjB9", 10)
	if req.After != "eyJpZCI6MTB9" || req.Before != "eyJpZCI6MjB9" {
		t.Errorf("cursors were not preserved verbatim: %+v", req)
	}
}

func TestCursorRequest_ComposesBaseRequest(t *testing.T) {
	t.Parallel()
	req := paginationkit.NewCursorPaginationRequest("a", "b", 7)
	if req.BaseRequest.Limit != 7 {
		t.Errorf("embedded BaseRequest.Limit: want 7, got %d", req.BaseRequest.Limit)
	}
}

func TestCursorRequest_JSONShape(t *testing.T) {
	t.Parallel()
	req := paginationkit.NewCursorPaginationRequest("after-token", "before-token", 15)
	buf, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got paginationkit.CursorRequest
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.After != "after-token" || got.Before != "before-token" || got.Limit != 15 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

// ---------- Offset response ----------------------------------------------

func TestNewOffsetPaginationResponse_TotalPreserved(t *testing.T) {
	t.Parallel()
	total := int64(123)
	resp := paginationkit.NewOffsetPaginationResponse(20, &total, true, false)
	if resp.Limit != 20 {
		t.Errorf("limit: want 20, got %d", resp.Limit)
	}
	if resp.HasNext != true || resp.HasPrevious != false {
		t.Errorf("flags wrong: %+v", resp.BaseResponse)
	}
	if resp.Total == nil || *resp.Total != 123 {
		t.Errorf("total not preserved: %+v", resp.Total)
	}
}

func TestNewOffsetPaginationResponse_NilTotalStaysNil(t *testing.T) {
	t.Parallel()
	resp := paginationkit.NewOffsetPaginationResponse(20, nil, false, false)
	if resp.Total != nil {
		t.Errorf("nil total must remain nil on response, got %+v", resp.Total)
	}
}

func TestNewOffsetPaginationResponse_LimitNormalised(t *testing.T) {
	t.Parallel()
	resp := paginationkit.NewOffsetPaginationResponse(9999, nil, false, false)
	if resp.Limit != paginationkit.MaxPaginationLimit {
		t.Errorf("response limit not normalised: want %d, got %d",
			paginationkit.MaxPaginationLimit, resp.Limit)
	}
	resp2 := paginationkit.NewOffsetPaginationResponse(0, nil, false, false)
	if resp2.Limit != paginationkit.DefaultPaginationLimit {
		t.Errorf("zero response limit not normalised: want %d, got %d",
			paginationkit.DefaultPaginationLimit, resp2.Limit)
	}
}

func TestOffsetResponse_TotalOmittedWhenNil(t *testing.T) {
	t.Parallel()
	// omitempty on a *int64 means a nil pointer drops out of the
	// wire shape entirely. Pin the contract so future refactors do
	// not silently flip it.
	resp := paginationkit.NewOffsetPaginationResponse(10, nil, false, false)
	buf, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(buf); got != `{"limit":10,"has_next":false,"has_previous":false}` {
		t.Errorf("wire shape mismatch: %s", got)
	}
}

func TestOffsetResponse_TotalPresentWhenSet(t *testing.T) {
	t.Parallel()
	total := int64(42)
	resp := paginationkit.NewOffsetPaginationResponse(10, &total, false, false)
	buf, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["total"]; !ok {
		t.Errorf("total must be present on the wire: %s", buf)
	}
}

// ---------- Cursor response ----------------------------------------------

func TestNewCursorPaginationResponse_CursorsPreserved(t *testing.T) {
	t.Parallel()
	next := "next-token"
	prev := "prev-token"
	resp := paginationkit.NewCursorPaginationResponse(20, &next, &prev, true, true)
	if resp.Limit != 20 {
		t.Errorf("limit: want 20, got %d", resp.Limit)
	}
	if !resp.HasNext || !resp.HasPrevious {
		t.Errorf("flags wrong: %+v", resp.BaseResponse)
	}
	if resp.NextCursor == nil || *resp.NextCursor != "next-token" {
		t.Errorf("next cursor not preserved: %+v", resp.NextCursor)
	}
	if resp.PrevCursor == nil || *resp.PrevCursor != "prev-token" {
		t.Errorf("prev cursor not preserved: %+v", resp.PrevCursor)
	}
}

func TestNewCursorPaginationResponse_NilCursorsStayNil(t *testing.T) {
	t.Parallel()
	resp := paginationkit.NewCursorPaginationResponse(20, nil, nil, false, false)
	if resp.NextCursor != nil {
		t.Errorf("nil next cursor must remain nil, got %+v", resp.NextCursor)
	}
	if resp.PrevCursor != nil {
		t.Errorf("nil prev cursor must remain nil, got %+v", resp.PrevCursor)
	}
}

func TestNewCursorPaginationResponse_LimitNormalised(t *testing.T) {
	t.Parallel()
	resp := paginationkit.NewCursorPaginationResponse(9999, nil, nil, false, false)
	if resp.Limit != paginationkit.MaxPaginationLimit {
		t.Errorf("response limit not normalised: want %d, got %d",
			paginationkit.MaxPaginationLimit, resp.Limit)
	}
}

func TestCursorResponse_CursorsOmittedWhenNil(t *testing.T) {
	t.Parallel()
	resp := paginationkit.NewCursorPaginationResponse(10, nil, nil, false, false)
	buf, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(buf); got != `{"limit":10,"has_next":false,"has_previous":false}` {
		t.Errorf("wire shape mismatch: %s", got)
	}
}

func TestCursorResponse_CursorsPresentWhenSet(t *testing.T) {
	t.Parallel()
	next := "n"
	prev := "p"
	resp := paginationkit.NewCursorPaginationResponse(10, &next, &prev, true, true)
	buf, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["next_cursor"]; !ok {
		t.Errorf("next_cursor must be present on the wire: %s", buf)
	}
	if _, ok := got["prev_cursor"]; !ok {
		t.Errorf("prev_cursor must be present on the wire: %s", buf)
	}
}

// ---------- Composition / embedding --------------------------------------

func TestResponses_ComposeBaseResponse(t *testing.T) {
	t.Parallel()
	// The concrete response types must embed BaseResponse; the
	// three BaseResponse fields must be reachable on them through
	// the embedded struct.
	off := paginationkit.NewOffsetPaginationResponse(15, nil, true, false)
	if off.Limit != 15 || !off.HasNext || off.HasPrevious {
		t.Errorf("OffsetResponse.BaseResponse fields wrong: %+v", off.BaseResponse)
	}

	cur := paginationkit.NewCursorPaginationResponse(15, nil, nil, true, true)
	if cur.Limit != 15 || !cur.HasNext || !cur.HasPrevious {
		t.Errorf("CursorResponse.BaseResponse fields wrong: %+v", cur.BaseResponse)
	}
}

// ---------- Sugar constructors ------------------------------------------

func TestFirstPage_IsEquivalentToOffsetZero(t *testing.T) {
	t.Parallel()
	cases := []int{0, -1, 1, 25, 9999}
	for _, limit := range cases {
		limit := limit
		t.Run("", func(t *testing.T) {
			t.Parallel()
			a := paginationkit.FirstPage(limit)
			b := paginationkit.NewOffsetPaginationRequest(0, limit)
			if a != b {
				t.Errorf("FirstPage(%d) != NewOffsetPaginationRequest(0, %d): %+v vs %+v",
					limit, limit, a, b)
			}
		})
	}
}

func TestEmptyOffsetResponse_AllFlagsFalse(t *testing.T) {
	t.Parallel()
	resp := paginationkit.EmptyOffsetResponse(20)
	if resp.Limit != 20 {
		t.Errorf("limit: want 20, got %d", resp.Limit)
	}
	if resp.HasNext || resp.HasPrevious {
		t.Errorf("empty response must have no flags set: %+v", resp.BaseResponse)
	}
	if resp.Total != nil {
		t.Errorf("empty response must have nil total, got %+v", resp.Total)
	}
}

func TestEmptyCursorResponse_AllFlagsFalse(t *testing.T) {
	t.Parallel()
	resp := paginationkit.EmptyCursorResponse(20)
	if resp.Limit != 20 {
		t.Errorf("limit: want 20, got %d", resp.Limit)
	}
	if resp.HasNext || resp.HasPrevious {
		t.Errorf("empty response must have no flags set: %+v", resp.BaseResponse)
	}
	if resp.NextCursor != nil || resp.PrevCursor != nil {
		t.Errorf("empty response must have nil cursors, got next=%+v prev=%+v",
			resp.NextCursor, resp.PrevCursor)
	}
}

// ---------- Defensive programming ----------------------------------------

// All constructors must be safe to call with the most adversarial
// inputs (huge negatives, INT_MAX). They must never panic.
func TestConstructors_NeverPanic(t *testing.T) {
	t.Parallel()
	inputs := []int{0, -1, -9999, 1, paginationkit.MaxPaginationLimit, paginationkit.MaxPaginationLimit + 1, 1 << 30}
	for _, n := range inputs {
		n := n
		t.Run("", func(t *testing.T) {
			t.Parallel()
			_ = paginationkit.NewOffsetPaginationRequest(n, n)
			_ = paginationkit.NewOffsetPaginationResponse(n, nil, false, false)
			_ = paginationkit.NewCursorPaginationRequest("a", "b", n)
			_ = paginationkit.NewCursorPaginationResponse(n, nil, nil, false, false)
			_ = paginationkit.FirstPage(n)
			_ = paginationkit.EmptyOffsetResponse(n)
			_ = paginationkit.EmptyCursorResponse(n)
		})
	}
}

// Output limits are always within [1, MaxPaginationLimit] after
// construction, regardless of what the caller passed in. This is the
// one invariant downstream code (databases, renderers) gets to rely
// on.
func TestConstructors_LimitAlwaysWithinBounds(t *testing.T) {
	t.Parallel()
	for _, n := range []int{-1, 0, 1, 100, 99999} {
		n := n
		t.Run("", func(t *testing.T) {
			t.Parallel()
			if got := paginationkit.NewOffsetPaginationRequest(0, n).Limit; got < 1 || got > paginationkit.MaxPaginationLimit {
				t.Errorf("offset req limit out of bounds: %d", got)
			}
			if got := paginationkit.NewOffsetPaginationResponse(n, nil, false, false).Limit; got < 1 || got > paginationkit.MaxPaginationLimit {
				t.Errorf("offset resp limit out of bounds: %d", got)
			}
			if got := paginationkit.NewCursorPaginationRequest("", "", n).Limit; got < 1 || got > paginationkit.MaxPaginationLimit {
				t.Errorf("cursor req limit out of bounds: %d", got)
			}
			if got := paginationkit.NewCursorPaginationResponse(n, nil, nil, false, false).Limit; got < 1 || got > paginationkit.MaxPaginationLimit {
				t.Errorf("cursor resp limit out of bounds: %d", got)
			}
		})
	}
}

// OffsetRequest.Offset is always >= 0 after construction.
func TestOffsetRequest_OffsetNeverNegative(t *testing.T) {
	t.Parallel()
	for _, n := range []int{-1, -100, 0, 1, 9999} {
		n := n
		t.Run("", func(t *testing.T) {
			t.Parallel()
			if got := paginationkit.NewOffsetPaginationRequest(n, 10).Offset; got < 0 {
				t.Errorf("offset went negative: %d", got)
			}
		})
	}
}