package responsekit

import (
	"encoding/json"
	"net/http"
)

// HTTPOK writes a 200 success response: {"data": data}.
func HTTPOK(w http.ResponseWriter, _ *http.Request, data any) {
	writeJSON(w, http.StatusOK, successEnvelope(data))
}

// HTTPCreated writes a 201 response.
func HTTPCreated(w http.ResponseWriter, _ *http.Request, data any) {
	writeJSON(w, http.StatusCreated, successEnvelope(data))
}

// HTTPAccepted writes a 202 response.
func HTTPAccepted(w http.ResponseWriter, _ *http.Request, data any) {
	writeJSON(w, http.StatusAccepted, successEnvelope(data))
}

// HTTPNoContent writes a 204 response with no body.
func HTTPNoContent(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// HTTPError writes an error response. Status is derived from err
// via statusCode; the body uses errorEnvelope.
func HTTPError(w http.ResponseWriter, _ *http.Request, err error) {
	writeJSON(w, statusCode(err), errorEnvelope(err))
}

// HTTPJSON is a passthrough for non-standard status codes / body
// shapes. Callers are responsible for matching the responsekit
// wire format if they want platform consistency.
func HTTPJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	writeJSON(w, status, body)
}

// writeJSON is the net/http-specific rendering helper. Unlike Gin
// and Fiber, net/http has no built-in JSON shortcut — this is the
// missing piece. Encode errors are intentionally dropped: by the
// time the marshal fails the headers are already on the wire, so
// there is no useful recovery path; callers that need write-error
// observability should use a custom middleware.
//
// json.Marshal (rather than json.NewEncoder) is used so the body
// does not include the trailing newline that Encoder.Encode appends
// — that keeps the wire output byte-for-byte identical to Gin's
// c.JSON and Fiber's c.JSON.
func writeJSON(w http.ResponseWriter, status int, body any) {
	buf, err := json.Marshal(body)
	if err != nil {
		// Headers not yet flushed — we can still return 500 with a
		// plain-text body. Production code should not reach here
		// because every responsekit body is statically typed.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf)
}