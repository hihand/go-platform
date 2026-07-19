package httperr_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/errkit/httperr"
)

func TestStatusCode_DefaultMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code errkit.Code
		want int
	}{
		{errkit.CodeInvalidArgument, http.StatusBadRequest},
		{errkit.CodeNotFound, http.StatusNotFound},
		{errkit.CodeAlreadyExists, http.StatusConflict},
		{errkit.CodeUnauthenticated, http.StatusUnauthorized},
		{errkit.CodePermissionDenied, http.StatusForbidden},
		{errkit.CodeUnavailable, http.StatusServiceUnavailable},
		{errkit.CodeDeadlineExceeded, http.StatusGatewayTimeout},
		{errkit.CodeCanceled, httperr.StatusClientClosedRequest},
		{errkit.CodeInternal, http.StatusInternalServerError},
		{errkit.CodeUnknown, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.code), func(t *testing.T) {
			t.Parallel()
			if got := httperr.StatusCode(errkit.New(errkit.WithCode(tc.code))); got != tc.want {
				t.Errorf("default: want %d, got %d", tc.want, got)
			}
		})
	}
}

func TestStatusCode_Fallbacks(t *testing.T) {
	t.Parallel()
	if got := httperr.StatusCode(nil); got != http.StatusInternalServerError {
		t.Errorf("nil err should default 500, got %d", got)
	}
	if got := httperr.StatusCode(errors.New("plain")); got != http.StatusInternalServerError {
		t.Errorf("non-errkit should default 500, got %d", got)
	}
	custom := errkit.Code("PAYMENT_REQUIRED") // unmapped
	if got := httperr.StatusCode(errkit.New(errkit.WithCode(custom))); got != http.StatusInternalServerError {
		t.Errorf("unmapped code should default 500, got %d", got)
	}
}

func TestMapper_Override(t *testing.T) {
	t.Parallel()
	m := httperr.NewMapper(map[errkit.Code]int{
		errkit.CodeInvalidArgument: http.StatusUnprocessableEntity,
	})
	if got := m.StatusCode(errkit.New(errkit.WithCode(errkit.CodeInvalidArgument))); got != http.StatusUnprocessableEntity {
		t.Errorf("override not applied: got %d", got)
	}
	// unaffected codes still hit the default map.
	if got := m.StatusCode(errkit.New(errkit.WithCode(errkit.CodeNotFound))); got != http.StatusNotFound {
		t.Errorf("default mapping broken by override: got %d", got)
	}
}

func TestNewMapper_OverrideIsCopied(t *testing.T) {
	t.Parallel()
	src := map[errkit.Code]int{errkit.CodeInternal: http.StatusBadGateway}
	m := httperr.NewMapper(src)
	src[errkit.CodeInternal] = http.StatusTeapot // mutate after construction
	if got := m.StatusCode(errkit.New(errkit.WithCode(errkit.CodeInternal))); got != http.StatusBadGateway {
		t.Errorf("mapper must not see post-construction mutation; got %d", got)
	}
}

func TestMapper_WalksCauseChain(t *testing.T) {
	t.Parallel()
	cause := errkit.NotFound("inner")
	wrapped := fmt.Errorf("ctx: %w", cause)
	if got := httperr.StatusCode(wrapped); got != http.StatusNotFound {
		t.Errorf("StatusCode should walk chain; got %d", got)
	}
}
