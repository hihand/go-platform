// Package paginationkit provides lightweight, transport-agnostic pagination
// models for the go-platform ecosystem.
//
// # Design contract
//
//   - The core package has zero dependencies on HTTP, gRPC, GraphQL,
//     databases, ORMs, or SQL. It is stdlib only and re-usable by any
//     transport or business layer.
//   - The package owns only pagination models, constructors, and
//     internal normalisation (validation that defaults sane values for
//     limit/offset/cursor). It does NOT do HTTP query parsing, JSON
//     encoding, or database slicing — those concerns live in adapter
//     packages and in caller code.
//   - Constructors are explicit and self-documenting (e.g.
//     NewOffsetPaginationRequest, NewCursorPaginationResponse). No
//     positional surprises, no global mutable state, no reflection.
//   - Values are normalised at construction time so downstream code can
//     trust what it reads (limit > 0, offset >= 0, etc.).
//   - Optional pagination fields (Total, NextCursor, PrevCursor) are
//     modelled as pointers so that "absent" is distinguishable from
//     "zero value".
//
// # File layout
//
// One concern per file. Constants and shared helpers live in common.go;
// the base/offset/cursor request models live in request.go; the
// matching response models live in response.go; the offset helpers
// live in offset.go; the cursor helpers live in cursor.go; and the
// sugar constructors live in new.go.
package paginationkit