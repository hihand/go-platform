package errkit_test

import (
	"errors"
	"testing"

	"github.com/hihand/go-platform/errkit"
)

func TestMetadataOf_DefaultsToEmpty(t *testing.T) {
	t.Parallel()
	if got := errkit.MetadataOf(nil); len(got) != 0 {
		t.Errorf("MetadataOf(nil) want empty map, got %v", got)
	}
	if got := errkit.MetadataOf(errors.New("plain")); len(got) != 0 {
		t.Errorf("MetadataOf(non-errkit) want empty map, got %v", got)
	}
	if got := errkit.MetadataOf(errkit.New()); got == nil {
		t.Errorf("MetadataOf(no metadata) want non-nil empty map")
	}
}

func TestMetadata_DefensiveCopy(t *testing.T) {
	t.Parallel()
	err := errkit.New(errkit.WithMetadata(map[string]any{
		"request_id": "abc-123",
		"retries":    3,
	}))
	md := errkit.MetadataOf(err)
	if md["request_id"] != "abc-123" {
		t.Errorf("metadata not present")
	}
	// mutate the returned map; original must not change.
	md["request_id"] = "tampered"
	if got := errkit.MetadataOf(err)["request_id"]; got != "abc-123" {
		t.Errorf("WithMetadata did not defensive-copy on construction; got %v", got)
	}
}

func TestMetadata_DefensiveCopy_OnWithMetadata(t *testing.T) {
	t.Parallel()
	src := map[string]any{"k": "v"}
	err := errkit.New(errkit.WithMetadata(src))
	// mutate after construction; must not leak into the error.
	src["k"] = "tampered"
	if got := errkit.MetadataOf(err)["k"]; got != "v" {
		t.Errorf("WithMetadata did not defensive-copy on input; got %v", got)
	}
}

func TestMetadata_EmptyMapIsNoOp(t *testing.T) {
	t.Parallel()
	err := errkit.New(errkit.WithMetadata(map[string]any{}))
	if got := errkit.MetadataOf(err); len(got) != 0 {
		t.Errorf("empty WithMetadata should leave metadata empty, got %v", got)
	}
}
