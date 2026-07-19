package responsekit_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"

	"github.com/hihand/go-platform/errkit"
	"github.com/hihand/go-platform/responsekit"
)

// Gin.SetMode is set once at process start to silence the startup
// banner. Safe to call multiple times but pointless after the first.
func init() { gin.SetMode(gin.TestMode) }

// ---------- Wire shape -----------------------------------------------------

func TestEnvelope_NilDataRendersNull(t *testing.T) {
	t.Parallel()
	buf, err := json.Marshal(responsekit.Envelope{Data: nil})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(buf); got != `{"data":null}` {
		t.Errorf("wire = %s, want %s", got, `{"data":null}`)
	}
}

func TestErrorEnvelope_Shape(t *testing.T) {
	t.Parallel()
	buf, err := json.Marshal(responsekit.ErrorEnvelope{
		Error: responsekit.ErrorBody{Code: "NOT_FOUND", Message: "missing"},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(buf); got != `{"error":{"code":"NOT_FOUND","message":"missing"}}` {
		t.Errorf("wire = %s", got)
	}
}

// ---------- Shared helpers -------------------------------------------------

// errorCases drives every Gin/Fiber/net/http error test matrix.
// All three adapters must produce the same status + body for the
// same input — that's the whole point of the shared helpers.
var errorCases = []errorCase{
	{"invalid_argument", errkit.InvalidArgument("bad input"), http.StatusBadRequest, "INVALID_ARGUMENT", "bad input"},
	{"unauthenticated", errkit.Unauthenticated("no token"), http.StatusUnauthorized, "UNAUTHENTICATED", "no token"},
	{"permission_denied", errkit.PermissionDenied("nope"), http.StatusForbidden, "PERMISSION_DENIED", "nope"},
	{"not_found", errkit.NotFound("user 42"), http.StatusNotFound, "NOT_FOUND", "user 42"},
	{"conflict", errkit.AlreadyExists("dup"), http.StatusConflict, "ALREADY_EXISTS", "dup"},
	{"internal", errkit.Internal("boom"), http.StatusInternalServerError, "INTERNAL", "boom"},
	{"unavailable", errkit.Unavailable("down"), http.StatusServiceUnavailable, "UNAVAILABLE", "down"},
	{"deadline_exceeded", errkit.DeadlineExceeded("timeout"), http.StatusGatewayTimeout, "DEADLINE_EXCEEDED", "timeout"},
	{
		"wrapped_errkit",
		errkit.Wrap(errors.New("redis: refused"),
			errkit.WithCode(errkit.CodeUnavailable),
			errkit.WithMessage("cache down")),
		http.StatusServiceUnavailable, "UNAVAILABLE", "cache down",
	},
	{"plain_error", errors.New("something broke"), http.StatusInternalServerError, "INTERNAL", "something broke"},
	{"nil_error", nil, http.StatusInternalServerError, "INTERNAL", ""},
}

// ---------- Gin adapter ----------------------------------------------------

func newGinCtx() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestGin_SuccessHelpers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		call   func(c *gin.Context)
		status int
		body   string
	}{
		{"ok", func(c *gin.Context) { responsekit.GinOK(c, map[string]string{"id": "u-1"}) }, http.StatusOK, `{"data":{"id":"u-1"}}`},
		{"ok_nil", func(c *gin.Context) { responsekit.GinOK(c, nil) }, http.StatusOK, `{"data":null}`},
		{"created", func(c *gin.Context) { responsekit.GinCreated(c, map[string]string{"id": "u-1"}) }, http.StatusCreated, `{"data":{"id":"u-1"}}`},
		{"accepted", func(c *gin.Context) { responsekit.GinAccepted(c, map[string]string{"job_id": "j-1"}) }, http.StatusAccepted, `{"data":{"job_id":"j-1"}}`},
		{"no_content", func(c *gin.Context) { responsekit.GinNoContent(c) }, http.StatusNoContent, ``},
		{"json_passthrough", func(c *gin.Context) { responsekit.GinJSON(c, http.StatusTeapot, map[string]string{"brew": "earl grey"}) }, http.StatusTeapot, `{"brew":"earl grey"}`},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c, w := newGinCtx()
			tc.call(c)
			if w.Code != tc.status {
				t.Errorf("status = %d, want %d", w.Code, tc.status)
			}
			if got := w.Body.String(); got != tc.body {
				t.Errorf("body = %s, want %s", got, tc.body)
			}
		})
	}
}

func TestGin_OK_SetsContentType(t *testing.T) {
	t.Parallel()
	c, w := newGinCtx()
	responsekit.GinOK(c, map[string]string{"id": "u-1"})
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestGin_Error(t *testing.T) {
	t.Parallel()
	for _, tc := range errorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c, w := newGinCtx()
			responsekit.GinError(c, tc.err)
			assertErrorResponse(t, w.Result(), tc)
		})
	}
}

// ---------- Fiber adapter --------------------------------------------------

func newFiberApp(handler fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/x", handler)
	app.Post("/x", handler)
	app.Delete("/x", handler)
	return app
}

func doFiber(t *testing.T, app *fiber.App, method, _ string, body io.Reader) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, "/x", body)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

func TestFiber_SuccessHelpers(t *testing.T) {
	t.Parallel()
	app := newFiberApp(func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet:
			return responsekit.FiberOK(c, map[string]string{"id": "u-1"})
		case fiber.MethodPost:
			return responsekit.FiberCreated(c, map[string]string{"id": "u-1"})
		case fiber.MethodDelete:
			return responsekit.FiberNoContent(c)
		}
		return nil
	})
	cases := []struct {
		name   string
		method string
		status int
		body   string
	}{
		{"ok", fiber.MethodGet, http.StatusOK, `{"data":{"id":"u-1"}}`},
		{"created", fiber.MethodPost, http.StatusCreated, `{"data":{"id":"u-1"}}`},
		{"no_content", fiber.MethodDelete, http.StatusNoContent, ``},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp := doFiber(t, app, tc.method, "/x", nil)
			assertResponse(t, resp, tc.status, tc.body)
		})
	}
}

func TestFiber_AcceptedAndJSON(t *testing.T) {
	t.Parallel()
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/accepted", func(c *fiber.Ctx) error {
		return responsekit.FiberAccepted(c, map[string]string{"job_id": "j-1"})
	})
	app.Get("/passthrough", func(c *fiber.Ctx) error {
		return responsekit.FiberJSON(c, fiber.StatusTeapot, map[string]string{"brew": "earl grey"})
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/accepted", nil), -1)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	assertResponse(t, resp, http.StatusAccepted, `{"data":{"job_id":"j-1"}}`)

	resp, err = app.Test(httptest.NewRequest(fiber.MethodGet, "/passthrough", nil), -1)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	assertResponse(t, resp, fiber.StatusTeapot, `{"brew":"earl grey"}`)
}

func TestFiber_Error(t *testing.T) {
	t.Parallel()
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/err", func(c *fiber.Ctx) error {
		// Pull the error from a header so the test matrix can drive
		// every case through one handler.
		key := c.Get("X-Test-Name")
		for _, tc := range errorCases {
			if tc.name == key {
				return responsekit.FiberError(c, tc.err)
			}
		}
		return responsekit.FiberError(c, errkit.Internal("unknown test"))
	})
	for _, tc := range errorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(fiber.MethodGet, "/err", nil)
			req.Header.Set("X-Test-Name", tc.name)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("test: %v", err)
			}
			assertErrorResponse(t, resp, tc)
		})
	}
}

// ---------- net/http adapter ----------------------------------------------

func newHTTPHandler(h http.HandlerFunc) http.Handler { return h }

func doHTTP(t *testing.T, h http.HandlerFunc, method, _ string, body io.Reader) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, "/", body)
	w := httptest.NewRecorder()
	h(w, req)
	return w.Result()
}

func TestHTTP_SuccessHelpers(t *testing.T) {
	t.Parallel()
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			responsekit.HTTPOK(w, r, map[string]string{"id": "u-1"})
		case http.MethodPost:
			responsekit.HTTPCreated(w, r, map[string]string{"id": "u-1"})
		case http.MethodPut:
			responsekit.HTTPAccepted(w, r, map[string]string{"job_id": "j-1"})
		case http.MethodDelete:
			responsekit.HTTPNoContent(w, r)
		}
	}
	cases := []struct {
		name   string
		method string
		status int
		body   string
	}{
		{"ok", http.MethodGet, http.StatusOK, `{"data":{"id":"u-1"}}`},
		{"created", http.MethodPost, http.StatusCreated, `{"data":{"id":"u-1"}}`},
		{"accepted", http.MethodPut, http.StatusAccepted, `{"data":{"job_id":"j-1"}}`},
		{"no_content", http.MethodDelete, http.StatusNoContent, ``},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp := doHTTP(t, handler, tc.method, "/", nil)
			assertResponse(t, resp, tc.status, tc.body)
		})
	}
}

func TestHTTP_JSONPassthrough(t *testing.T) {
	t.Parallel()
	handler := func(w http.ResponseWriter, r *http.Request) {
		responsekit.HTTPJSON(w, r, http.StatusTeapot, map[string]string{"brew": "earl grey"})
	}
	resp := doHTTP(t, handler, http.MethodGet, "/", nil)
	assertResponse(t, resp, http.StatusTeapot, `{"brew":"earl grey"}`)
}

func TestHTTP_SetsContentType(t *testing.T) {
	t.Parallel()
	handler := func(w http.ResponseWriter, r *http.Request) {
		responsekit.HTTPOK(w, r, map[string]string{"id": "u-1"})
	}
	resp := doHTTP(t, handler, http.MethodGet, "/", nil)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
}

func TestHTTP_Error(t *testing.T) {
	t.Parallel()
	handler := func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Test-Name")
		for _, tc := range errorCases {
			if tc.name == key {
				responsekit.HTTPError(w, r, tc.err)
				return
			}
		}
		responsekit.HTTPError(w, r, errkit.Internal("unknown test"))
	}
	for _, tc := range errorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Test-Name", tc.name)
			w := httptest.NewRecorder()
			handler(w, req)
			assertErrorResponse(t, w.Result(), tc)
		})
	}
}

// ---------- Shared assertions ---------------------------------------------

// errorBody matches responsekit.ErrorEnvelope for assertions.
type errorBody struct {
	Error struct {
		Code, Message string
	} `json:"error"`
}

// assertResponse checks status code and body string. Used by every
// adapter's success tests.
func assertResponse(t *testing.T, resp *http.Response, status int, body string) {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != status {
		t.Errorf("status = %d, want %d", resp.StatusCode, status)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != body {
		t.Errorf("body = %s, want %s", got, body)
	}
}

// errorCase mirrors one row of errorCases. Asserting against the
// concrete case (instead of looking it up by code) lets the matrix
// contain duplicate codes — wrapped_errkit and unavailable both
// render "UNAVAILABLE", but only the right row is asserted.
type errorCase struct {
	name   string
	err    error
	status int
	code   string
	msg    string
}

// assertErrorResponse decodes the error envelope and verifies the
// status code + code/message fields against the supplied case.
func assertErrorResponse(t *testing.T, resp *http.Response, want errorCase) {
	t.Helper()
	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var got errorBody
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("decode %q: %v", buf, err)
	}
	if resp.StatusCode != want.status {
		t.Errorf("[%s] status = %d, want %d (code=%s)", want.name, resp.StatusCode, want.status, want.code)
	}
	if got.Error.Code != want.code {
		t.Errorf("[%s] code = %q, want %q", want.name, got.Error.Code, want.code)
	}
	if got.Error.Message != want.msg {
		t.Errorf("[%s] message = %q, want %q", want.name, got.Error.Message, want.msg)
	}
}