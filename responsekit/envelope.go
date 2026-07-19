// Package responsekit is a small, framework-agnostic HTTP response
// helper for the go-platform ecosystem.
//
// The package owns the wire format. Each adapter (gin.go, fiber.go,
// nethttp.go) is a thin translation that reuses the same three
// helpers from common.go, so every adapter produces identical JSON
// for the same input.
//
// Wire shapes:
//
//	{"data": ...}
//	{"error": {"code": "...", "message": "..."}}
//
// Things that are intentionally not present: success/status flags,
// timestamps, request IDs, trace IDs, pagination metadata, and
// the original request path. Those concerns live in middleware and
// structured logging, not in response bodies.
package responsekit

// Envelope is the wire shape of a successful response. Data is
// intentionally typed as any so callers can return arbitrary Go
// values (single objects, arrays, scalars, maps); encoding/json
// renders whatever is provided. A nil data renders as null, not
// omitted, so the wire shape stays predictable.
type Envelope struct {
	Data any `json:"data"`
}

// ErrorEnvelope is the wire shape of a failed response.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody is the inner shape of the "error" object on the wire.
// Code is the stable, machine-friendly identifier; Message is the
// human-readable description.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}