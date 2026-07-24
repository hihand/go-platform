package contextkit_test

import (
	"context"
	"fmt"

	"github.com/hihand/go-platform/contextkit"
)

// ExampleWithRequest shows how a transport edge stores a request
// identifier on the context and how a downstream layer reads it
// back later. The returned Request is a copy, so mutating it has no
// effect on what other layers see.
func ExampleWithRequest() {
	ctx := context.Background()
	ctx = contextkit.WithRequest(ctx, contextkit.Request{
		RequestID: "req-abc",
		TraceID:   "trace-001",
		SpanID:    "span-002",
	})

	req := contextkit.GetRequest(ctx)
	fmt.Println(req.RequestID)
	fmt.Println(req.TraceID)
	fmt.Println(req.SpanID)
	// Output:
	// req-abc
	// trace-001
	// span-002
}

// ExampleWithIdentity shows how to attach an auth-derived struct to
// a context. The package has no opinion on the concrete identity
// type — it could just as well be a JWT claims struct, a merchant
// record, or a principal. Here we reuse the claims type defined in
// contextkit_test.go so the example shares one definition with the
// test suite.
func ExampleWithIdentity() {
	ctx := context.Background()
	ctx = contextkit.WithIdentity(ctx, claims{
		UserID:     "u-1",
		MerchantID: "m-7",
	})

	got, ok := contextkit.Identity[claims](ctx)
	fmt.Println(ok)
	fmt.Println(got.UserID)
	fmt.Println(got.MerchantID)
	// Output:
	// true
	// u-1
	// m-7
}

// ExampleIdentity demonstrates that reading a missing identity
// surfaces as a clean "absent" signal instead of a panic. This is
// the contract that lets callers default to anonymous behaviour at
// the top of a handler without having to remember whether
// middleware ran.
func ExampleIdentity() {
	got, ok := contextkit.Identity[claims](context.Background())
	fmt.Println(ok)
	fmt.Println(got.UserID == "" && got.MerchantID == "" && len(got.Roles) == 0)
	// Output:
	// false
	// true
}
