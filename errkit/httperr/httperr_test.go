package httperr_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/errkit/httperr"
)

// defaultMapping exhaustively pins every Code the library maps by default.
// If you add a Code or change a mapping, add the row here so the wire
// contract stays explicit.
func TestStatusCode_DefaultMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code errkit.Code
		want int
	}{
		// Transport / lifecycle
		{errkit.CodeCanceled, httperr.StatusClientClosedRequest},
		{errkit.CodeDeadlineExceeded, http.StatusGatewayTimeout},
		{errkit.CodeRequestTimeout, http.StatusRequestTimeout},

		// Client errors
		{errkit.CodeInvalidArgument, http.StatusBadRequest},
		{errkit.CodeUnauthenticated, http.StatusUnauthorized},
		{errkit.CodePermissionDenied, http.StatusForbidden},
		{errkit.CodeNotFound, http.StatusNotFound},
		{errkit.CodeMethodNotAllowed, http.StatusMethodNotAllowed},
		{errkit.CodeNotAcceptable, http.StatusNotAcceptable},
		{errkit.CodeConflict, http.StatusConflict},
		{errkit.CodeAlreadyExists, http.StatusConflict},
		{errkit.CodeGone, http.StatusGone},
		{errkit.CodeLengthRequired, http.StatusLengthRequired},
		{errkit.CodePreconditionFailed, http.StatusPreconditionFailed},
		{errkit.CodePayloadTooLarge, http.StatusRequestEntityTooLarge},
		{errkit.CodeURITooLong, http.StatusRequestURITooLong},
		{errkit.CodeUnsupportedMediaType, http.StatusUnsupportedMediaType},
		{errkit.CodeRangeNotSatisfiable, http.StatusRequestedRangeNotSatisfiable},
		{errkit.CodeExpectationFailed, http.StatusExpectationFailed},
		{errkit.CodeMisdirectedRequest, http.StatusMisdirectedRequest},
		{errkit.CodeUnprocessableEntity, http.StatusUnprocessableEntity},
		{errkit.CodeLocked, http.StatusLocked},
		{errkit.CodeFailedDependency, http.StatusFailedDependency},
		{errkit.CodeTooManyRequests, http.StatusTooManyRequests},
		{errkit.CodeRequestHeaderFieldsTooLarge, http.StatusRequestHeaderFieldsTooLarge},
		{errkit.CodeUnavailableForLegalReasons, http.StatusUnavailableForLegalReasons},

		// Server errors
		{errkit.CodeInternal, http.StatusInternalServerError},
		{errkit.CodeNotImplemented, http.StatusNotImplemented},
		{errkit.CodeBadGateway, http.StatusBadGateway},
		{errkit.CodeUnavailable, http.StatusServiceUnavailable},
		{errkit.CodeDataLoss, http.StatusInsufficientStorage},
		{errkit.CodeNetworkAuthenticationRequired, http.StatusNetworkAuthenticationRequired},

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
	// Unmapped custom codes (and built-ins intentionally left out of the
	// default table) fall back to 500 — the choice belongs to the app.
	for _, code := range []errkit.Code{
		errkit.CodeDuplicate,
		errkit.CodePaymentRequired,
		errkit.CodeUpgradeRequired,
		errkit.Code("CUSTOM_CODE"),
	} {
		if got := httperr.StatusCode(errkit.New(errkit.WithCode(code))); got != http.StatusInternalServerError {
			t.Errorf("unmapped code %q should default 500, got %d", code, got)
		}
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

// Mapper is free to introduce mappings for codes the default table skips
// (CodeDuplicate, CodePaymentRequired, …). Exercise that path so a custom
// 402/409 mapping never silently regresses.
func TestMapper_AddsNewEntries(t *testing.T) {
	t.Parallel()
	m := httperr.NewMapper(map[errkit.Code]int{
		errkit.CodeDuplicate:     http.StatusConflict,
		errkit.CodePaymentRequired: http.StatusPaymentRequired,
	})
	if got := m.StatusCode(errkit.New(errkit.WithCode(errkit.CodeDuplicate))); got != http.StatusConflict {
		t.Errorf("CodeDuplicate override not applied: got %d", got)
	}
	if got := m.StatusCode(errkit.New(errkit.WithCode(errkit.CodePaymentRequired))); got != http.StatusPaymentRequired {
		t.Errorf("CodePaymentRequired override not applied: got %d", got)
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
