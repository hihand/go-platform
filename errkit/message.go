package errkit

// Message returns the human-readable message attached to this error. It is
// always safe to call on an *impl; an empty string is a valid Message for
// callers that only care about Code and Metadata.
func (i *impl) Message() string {
	return i.message
}

// MessageOf extracts the Message from an arbitrary error, walking the cause
// chain via FromError. Returns the empty string when no errkit error is
// present, mirroring the behaviour of Message() on a default-constructed
// impl.
func MessageOf(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := FromError(err); ok {
		return e.Message()
	}
	return ""
}
