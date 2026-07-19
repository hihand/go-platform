package errkit

import "maps"

// MetadataAccessor is the optional interface implemented by *impl so callers
// can read attached metadata without enlarging the canonical Error interface.
// Helpers in this package, and any adapter, tolerate the absence of this
// method on foreign errors.
type MetadataAccessor interface {
	// Metadata returns a defensive copy of the metadata map. Mutating the
	// returned map has no effect on the error.
	Metadata() map[string]any
}

// Metadata returns a defensive copy of the attached metadata. The returned
// map is always non-nil and safe to read; it is empty when WithMetadata was
// not supplied.
func (i *impl) Metadata() map[string]any {
	if i.metadata == nil {
		return map[string]any{}
	}
	return maps.Clone(i.metadata)
}

// MetadataOf extracts the metadata from an arbitrary error, walking the
// cause chain via FromError. Returns an empty (non-nil) map when no
// errkit error is present or when the errkit error carries no metadata.
func MetadataOf(err error) map[string]any {
	if err == nil {
		return map[string]any{}
	}
	if m, ok := FromError(err); ok {
		if acc, ok := m.(MetadataAccessor); ok {
			return acc.Metadata()
		}
	}
	return map[string]any{}
}