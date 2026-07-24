package contextkit_test

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/hihand/go-platform/contextkit"
)

// ---------- shared types -------------------------------------------------

// claims is a stand-in for whatever auth-shaped struct a caller would
// store under the identity key. The test suite must exercise a real
// struct (not just a primitive) because the generic identity API is
// only useful for struct payloads.
type claims struct {
	UserID      string
	MerchantID  string
	Roles       []string
	permissions map[string]bool
}

// otherClaims is a different struct type with overlapping field
// names. It is used to assert that asking for a different T at read
// time surfaces as "missing", not as a panic.
type otherClaims struct {
	UserID string
}

// ---------- Request -------------------------------------------------------

func TestRequest_NilContext(t *testing.T) {
	t.Parallel()

	// Setter must not panic on a nil context and must return a
	// usable context.
	ctx := contextkit.WithRequest(nil, contextkit.Request{
		RequestID: "req-1",
	})
	if ctx == nil {
		t.Fatal("WithRequest(nil, ...) returned nil context")
	}

	// Getter must not panic on a nil context and must return the
	// zero value.
	if got := contextkit.GetRequest(nil); got != (contextkit.Request{}) {
		t.Errorf("GetRequest(nil) = %+v, want zero value", got)
	}
}

func TestRequest_MissingValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	if got := contextkit.GetRequest(ctx); got != (contextkit.Request{}) {
		t.Errorf("GetRequest on empty ctx = %+v, want zero value", got)
	}
}

func TestRequest_RoundTrip(t *testing.T) {
	t.Parallel()

	want := contextkit.Request{
		RequestID: "req-abc",
		TraceID:   "trace-001",
		SpanID:    "span-002",
	}
	ctx := contextkit.WithRequest(context.Background(), want)

	got := contextkit.GetRequest(ctx)
	if got != want {
		t.Errorf("GetRequest = %+v, want %+v", got, want)
	}
}

func TestRequest_Overwrite(t *testing.T) {
	t.Parallel()

	first := contextkit.Request{RequestID: "req-1", TraceID: "t-1"}
	second := contextkit.Request{RequestID: "req-2", TraceID: "t-2", SpanID: "s-2"}

	parent := contextkit.WithRequest(context.Background(), first)
	child := contextkit.WithRequest(parent, second)

	if got := contextkit.GetRequest(child); got != second {
		t.Errorf("child = %+v, want %+v", got, second)
	}
	// Parent must remain untouched — overwriting on the child
	// does not mutate ancestors.
	if got := contextkit.GetRequest(parent); got != first {
		t.Errorf("parent = %+v, want %+v (immutability broken)", got, first)
	}
}

func TestRequest_ReturnedValueIsCopy(t *testing.T) {
	t.Parallel()

	stored := contextkit.Request{RequestID: "req-1"}
	ctx := contextkit.WithRequest(context.Background(), stored)

	// Mutating the struct returned by GetRequest must not affect
	// what is stored in the context.
	got := contextkit.GetRequest(ctx)
	got.RequestID = "mutated"
	got.TraceID = "mutated-trace"

	again := contextkit.GetRequest(ctx)
	if again != stored {
		t.Errorf("context value changed after caller mutation: got %+v, want %+v", again, stored)
	}
}

func TestRequest_PartialFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  contextkit.Request
	}{
		{name: "only request id", req: contextkit.Request{RequestID: "req-1"}},
		{name: "only trace id", req: contextkit.Request{TraceID: "t-1"}},
		{name: "only span id", req: contextkit.Request{SpanID: "s-1"}},
		{name: "all empty", req: contextkit.Request{}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := contextkit.WithRequest(context.Background(), tc.req)
			if got := contextkit.GetRequest(ctx); got != tc.req {
				t.Errorf("round-trip mismatch: got %+v, want %+v", got, tc.req)
			}
		})
	}
}

// ---------- Identity: generic semantics ---------------------------------

func TestIdentity_NilContext(t *testing.T) {
	t.Parallel()

	// Setter must not panic.
	ctx := contextkit.WithIdentity(nil, claims{UserID: "u-1"})
	if ctx == nil {
		t.Fatal("WithIdentity(nil, ...) returned nil context")
	}

	// Getter must not panic and must report absence.
	got, ok := contextkit.Identity[claims](nil)
	if ok {
		t.Errorf("Identity[claims](nil) ok=true, want false")
	}
	if !reflect.DeepEqual(got, claims{}) {
		t.Errorf("Identity[claims](nil) value = %+v, want zero", got)
	}
}

func TestIdentity_MissingValue(t *testing.T) {
	t.Parallel()

	got, ok := contextkit.Identity[claims](context.Background())
	if ok {
		t.Errorf("ok = true, want false")
	}
	if !reflect.DeepEqual(got, claims{}) {
		t.Errorf("value = %+v, want zero", got)
	}
}

func TestIdentity_RoundTrip(t *testing.T) {
	t.Parallel()

	want := claims{
		UserID:     "u-1",
		MerchantID: "m-7",
		Roles:      []string{"admin", "billing"},
	}
	ctx := contextkit.WithIdentity(context.Background(), want)

	got, ok := contextkit.Identity[claims](ctx)
	if !ok {
		t.Fatal("Identity returned ok=false on present value")
	}
	if got.UserID != want.UserID || got.MerchantID != want.MerchantID {
		t.Errorf("Identity mismatch: got %+v, want %+v", got, want)
	}
	if len(got.Roles) != len(want.Roles) {
		t.Fatalf("Roles length = %d, want %d", len(got.Roles), len(want.Roles))
	}
	for i := range want.Roles {
		if got.Roles[i] != want.Roles[i] {
			t.Errorf("Roles[%d] = %q, want %q", i, got.Roles[i], want.Roles[i])
		}
	}
}

func TestIdentity_WrongType(t *testing.T) {
	t.Parallel()

	// Store as claims; ask for otherClaims. The type check must
	// fail silently and return (zero, false) — never panic.
	ctx := contextkit.WithIdentity(context.Background(), claims{
		UserID: "u-1",
	})

	got, ok := contextkit.Identity[otherClaims](ctx)
	if ok {
		t.Errorf("Identity[otherClaims] ok=true, want false (stored type was claims)")
	}
	if got != (otherClaims{}) {
		t.Errorf("value = %+v, want zero otherClaims", got)
	}

	// Sanity: the original T still retrieves the same value.
	right, ok := contextkit.Identity[claims](ctx)
	if !ok || right.UserID != "u-1" {
		t.Errorf("Identity[claims] broken after wrong-type query: got %+v, ok=%v", right, ok)
	}
}

func TestIdentity_Overwrite(t *testing.T) {
	t.Parallel()

	first := claims{UserID: "u-1"}
	second := claims{UserID: "u-2", MerchantID: "m-2"}

	parent := contextkit.WithIdentity(context.Background(), first)
	child := contextkit.WithIdentity(parent, second)

	got, ok := contextkit.Identity[claims](child)
	if !ok || got.UserID != "u-2" || got.MerchantID != "m-2" {
		t.Errorf("child = %+v (ok=%v), want u-2/m-2", got, ok)
	}
	// Parent immutability — must still see the original value.
	parentGot, ok := contextkit.Identity[claims](parent)
	if !ok || parentGot.UserID != "u-1" || parentGot.MerchantID != "" {
		t.Errorf("parent = %+v (ok=%v), want u-1 only (immutability broken)", parentGot, ok)
	}
}

func TestIdentity_PointerPayload(t *testing.T) {
	t.Parallel()

	// Storing and retrieving via pointer types must round-trip
	// exactly. This catches a class of "value vs. pointer"
	// interface-boxing bugs that the generic Identity API could
	// otherwise introduce.
	c := &claims{UserID: "u-1", MerchantID: "m-7"}
	ctx := contextkit.WithIdentity(context.Background(), c)

	got, ok := contextkit.Identity[*claims](ctx)
	if !ok {
		t.Fatal("Identity[*claims] ok=false on present value")
	}
	if got != c {
		t.Errorf("Identity[*claims] returned a different pointer: got %p, want %p", got, c)
	}

	// Asking for the value type (not pointer) must report
	// absence — no implicit dereferencing, no panic.
	_, ok = contextkit.Identity[claims](ctx)
	if ok {
		t.Errorf("Identity[claims] on *claims store: ok=true, want false")
	}
}

// ---------- multi-layer context ------------------------------------------

func TestLayers_RequestAndIdentityCoexist(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = contextkit.WithRequest(ctx, contextkit.Request{RequestID: "req-1"})
	ctx = contextkit.WithIdentity(ctx, claims{UserID: "u-1"})

	req := contextkit.GetRequest(ctx)
	ident, ok := contextkit.Identity[claims](ctx)

	if req.RequestID != "req-1" {
		t.Errorf("Request.RequestID = %q, want req-1", req.RequestID)
	}
	if !ok || ident.UserID != "u-1" {
		t.Errorf("Identity.UserID = %q (ok=%v), want u-1", ident.UserID, ok)
	}

	// Overwriting only one side must not affect the other.
	ctx = contextkit.WithRequest(ctx, contextkit.Request{RequestID: "req-2"})
	if got := contextkit.GetRequest(ctx).RequestID; got != "req-2" {
		t.Errorf("Request.RequestID after overwrite = %q, want req-2", got)
	}
	if got, ok := contextkit.Identity[claims](ctx); !ok || got.UserID != "u-1" {
		t.Errorf("Identity lost after Request overwrite: got %+v (ok=%v)", got, ok)
	}
}

func TestLayers_DeepCancellation(t *testing.T) {
	t.Parallel()

	// Cancellation propagation is stdlib behaviour, but we
	// confirm the package's helpers play nicely with it: a
	// context cancelled after WithRequest still exposes the
	// stored Request via the package getter, while the inner
	// Err() reports the cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	ctx = contextkit.WithRequest(ctx, contextkit.Request{RequestID: "req-1"})
	cancel()

	if err := ctx.Err(); err == nil {
		t.Fatal("ctx.Err() = nil after cancel, want non-nil")
	}
	if got := contextkit.GetRequest(ctx); got.RequestID != "req-1" {
		t.Errorf("GetRequest after cancel = %+v, want req-1", got)
	}
}

// ---------- parallel safety ----------------------------------------------

func TestParallel_NoSharedState(t *testing.T) {
	t.Parallel()

	// Drive a tight concurrent workload that exercises every API
	// in goroutines. The package has no globals and no shared
	// state, so a successful run with -race is the strict signal.
	const goroutines = 64
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				ctx := context.Background()
				ctx = contextkit.WithRequest(ctx, contextkit.Request{
					RequestID: reqIDFor(g, i),
				})
				ctx = contextkit.WithIdentity(ctx, claims{
					UserID: userIDFor(g, i),
				})

				if got := contextkit.GetRequest(ctx).RequestID; got != reqIDFor(g, i) {
					t.Errorf("goroutine %d iter %d: Request.RequestID = %q, want %q",
						g, i, got, reqIDFor(g, i))
				}
				if got, ok := contextkit.Identity[claims](ctx); !ok || got.UserID != userIDFor(g, i) {
					t.Errorf("goroutine %d iter %d: Identity.UserID = %q (ok=%v), want %q",
						g, i, got.UserID, ok, userIDFor(g, i))
				}
			}
		}()
	}
	wg.Wait()
}

func reqIDFor(g, i int) string  { return "req-" + itoa(g) + "-" + itoa(i) }
func userIDFor(g, i int) string { return "u-" + itoa(g) + "-" + itoa(i) }

// itoa is a tiny non-allocating integer printer used only by the
// parallel test helpers. It mirrors the one in main.go.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
