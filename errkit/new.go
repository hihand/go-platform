package errkit

// impl is the sole concrete Error implementation in this package. It is
// intentionally unexported; callers interact with it only through the
// Error interface and the helpers in code.go / message.go / metadata.go.
//
// The struct is kept tiny on purpose: every field here is part of the public
// contract, and every option mutates exactly one of them. The stack field is
// intentionally typed as any so that the future stack-trace contract can be
// shaped without breaking source compatibility for v1 users.
type impl struct {
	code     Code
	message  string
	cause    error
	metadata map[string]any
	stack    []uintptr
}

// defaultCode is the Code returned when New is called without WithCode. It
// is exposed to the package so a handful of helpers (CodeOf, FromError) can
// use the same value consistently.
const defaultCode Code = CodeUnknown

// New builds an errkit Error from the supplied options. Message is *not*
// positional; use WithMessage alongside WithCode, WithCause, WithMetadata,
// and/or WithStack.
//
// A zero-value call is allowed and produces an Error with default code and
// empty message:
//
//	errkit.New()                            // => CodeUnknown, ""
//	errkit.New(WithCode(errkit.CodeInternal)) // Code only
//
// Sugar constructors below cover the common cases for callers that don't
// need full control.
func New(opts ...Option) Error {
	err := &impl{code: defaultCode}
	for _, opt := range opts {
		opt(err)
	}

	return err
}

// Wrap attaches errkit attributes to an existing error and returns the
// resulting Error. If err is nil, Wrap returns nil — mirroring
// fmt.Errorf("%w", nil) semantics so the result can be returned directly
// from a function without nil checks:
//
//	if err != nil {
//	    return errkit.Wrap(err, errkit.WithCode(errkit.CodeInternal))
//	}
//
// err is exposed via Unwrap(); errors.Is/As continue to behave as expected.
func Wrap(err error, opts ...Option) Error {
	if err == nil {
		return nil
	}
	e := &impl{code: defaultCode, cause: err}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// =============================================================================
// Sugar constructors
//
// One sugar per Code that is meant to be raised directly. Codes that are
// intentionally *not* sugared (e.g. CodeDuplicate, CodePaymentRequired,
// CodeUpgradeRequired) require `New(WithCode(...), WithMessage(...))` — this
// forces the caller to be deliberate about the wire choice.
// =============================================================================

func NotFound(msg string) Error {
	return New(WithCode(CodeNotFound), WithMessage(msg))
}

func InvalidArgument(msg string) Error {
	return New(WithCode(CodeInvalidArgument), WithMessage(msg))
}

func AlreadyExists(msg string) Error {
	return New(WithCode(CodeAlreadyExists), WithMessage(msg))
}

func Conflict(msg string) Error {
	return New(WithCode(CodeConflict), WithMessage(msg))
}

func Unauthenticated(msg string) Error {
	return New(WithCode(CodeUnauthenticated), WithMessage(msg))
}

func PermissionDenied(msg string) Error {
	return New(WithCode(CodePermissionDenied), WithMessage(msg))
}

func Unavailable(msg string) Error {
	return New(WithCode(CodeUnavailable), WithMessage(msg))
}

func DeadlineExceeded(msg string) Error {
	return New(WithCode(CodeDeadlineExceeded), WithMessage(msg))
}

func RequestTimeout(msg string) Error {
	return New(WithCode(CodeRequestTimeout), WithMessage(msg))
}

func Canceled(msg string) Error {
	return New(WithCode(CodeCanceled), WithMessage(msg))
}

func TooManyRequests(msg string) Error {
	return New(WithCode(CodeTooManyRequests), WithMessage(msg))
}

func UnprocessableEntity(msg string) Error {
	return New(WithCode(CodeUnprocessableEntity), WithMessage(msg))
}

func PayloadTooLarge(msg string) Error {
	return New(WithCode(CodePayloadTooLarge), WithMessage(msg))
}

func MethodNotAllowed(msg string) Error {
	return New(WithCode(CodeMethodNotAllowed), WithMessage(msg))
}

func NotAcceptable(msg string) Error {
	return New(WithCode(CodeNotAcceptable), WithMessage(msg))
}

func Gone(msg string) Error {
	return New(WithCode(CodeGone), WithMessage(msg))
}

func BadGateway(msg string) Error {
	return New(WithCode(CodeBadGateway), WithMessage(msg))
}

func NotImplemented(msg string) Error {
	return New(WithCode(CodeNotImplemented), WithMessage(msg))
}

func Internal(msg string) Error {
	return New(WithCode(CodeInternal), WithMessage(msg))
}

func DataLoss(msg string) Error {
	return New(WithCode(CodeDataLoss), WithMessage(msg))
}

func PreconditionFailed(msg string) Error {
	return New(WithCode(CodePreconditionFailed), WithMessage(msg))
}

func UnsupportedMediaType(msg string) Error {
	return New(WithCode(CodeUnsupportedMediaType), WithMessage(msg))
}
