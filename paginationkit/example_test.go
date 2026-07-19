package paginationkit_test

import (
	"fmt"

	"github.com/hihand/go-platform/paginationkit"
)

// ExampleNewOffsetPaginationRequest shows how to build an offset
// request from raw transport input. Note that the raw limit 9999 is
// clamped to MaxPaginationLimit (100) and the raw offset -5 is
// clamped to 0, so downstream code can rely on the values it reads.
func ExampleNewOffsetPaginationRequest() {
	req := paginationkit.NewOffsetPaginationRequest(-5, 9999)
	fmt.Println(req.Offset)
	fmt.Println(req.Limit)
	// Output:
	// 0
	// 100
}

// ExampleNewOffsetPaginationResponse shows the canonical "I queried,
// there were 137 matching items, this is the first page, there is a
// next page, there is no previous page" response.
func ExampleNewOffsetPaginationResponse() {
	total := int64(137)
	resp := paginationkit.NewOffsetPaginationResponse(20, &total, true, false)
	fmt.Println(resp.Limit)
	fmt.Println(resp.HasNext)
	fmt.Println(resp.HasPrevious)
	fmt.Println(*resp.Total)
	// Output:
	// 20
	// true
	// false
	// 137
}

// ExampleNewOffsetPaginationResponse_nilTotal shows the same response
// shape with Total left nil — i.e. the service layer chose not to run
// COUNT(*). On the wire, a nil Total is omitted entirely.
func ExampleNewOffsetPaginationResponse_nilTotal() {
	resp := paginationkit.NewOffsetPaginationResponse(20, nil, false, false)
	fmt.Println(resp.Total == nil)
	// Output:
	// true
}

// ExampleNewCursorPaginationRequest shows a forward-paging request.
// The After cursor is opaque to paginationkit and is stored verbatim;
// the limit is normalised.
func ExampleNewCursorPaginationRequest() {
	req := paginationkit.NewCursorPaginationRequest("eyJpZCI6MTB9", "", 25)
	fmt.Println(req.After)
	fmt.Println(req.Before)
	fmt.Println(req.Limit)
	// Output:
	// eyJpZCI6MTB9
	//
	// 25
}

// ExampleNewCursorPaginationResponse shows a page in the middle of
// a result set: there is a next page and a previous page, and the
// response carries the cursors the caller needs to fetch them.
func ExampleNewCursorPaginationResponse() {
	next := "eyJpZCI6MjB9"
	prev := "eyJpZCI6MTAifQ"
	resp := paginationkit.NewCursorPaginationResponse(10, &next, &prev, true, true)
	fmt.Println(resp.Limit)
	fmt.Println(resp.HasNext)
	fmt.Println(resp.HasPrevious)
	fmt.Println(*resp.NextCursor)
	fmt.Println(*resp.PrevCursor)
	// Output:
	// 10
	// true
	// true
	// eyJpZCI6MjB9
	// eyJpZCI6MTAifQ
}

// ExampleFirstPage is a sugar shortcut for the most common offset
// case: "give me the first page of N items". Equivalent to
// NewOffsetPaginationRequest(0, n) but reads more naturally.
func ExampleFirstPage() {
	req := paginationkit.FirstPage(15)
	fmt.Println(req.Offset)
	fmt.Println(req.Limit)
	// Output:
	// 0
	// 15
}

// ExampleEmptyOffsetResponse is the canonical "no results" shape for
// offset pagination. Total is nil, both flags are false.
func ExampleEmptyOffsetResponse() {
	resp := paginationkit.EmptyOffsetResponse(20)
	fmt.Println(resp.Limit)
	fmt.Println(resp.HasNext)
	fmt.Println(resp.HasPrevious)
	fmt.Println(resp.Total == nil)
	// Output:
	// 20
	// false
	// false
	// true
}