package domain

import "fmt"

// ErrorCode is a stable machine-readable error code returned to clients.
type ErrorCode string

const (
	// CodeNotFound means the requested resource does not exist.
	CodeNotFound ErrorCode = "not_found"
	// CodeInvalidInput means the request payload failed validation.
	CodeInvalidInput ErrorCode = "invalid_input"
	// CodeConflict means the request violates an invariant of the domain
	// (for example an illegal state-machine transition or a restore
	// confirmation before the recovery condition is met).
	CodeConflict ErrorCode = "conflict"
	// CodeInternal is used for unexpected failures.
	CodeInternal ErrorCode = "internal"
	// CodeUnauthorized is reserved for future operator authentication.
	CodeUnauthorized ErrorCode = "unauthorized"
)

// Error is a domain error carrying a stable code, a human-readable message
// and the underlying cause (if any).
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped cause so errors.Is/As work.
func (e *Error) Unwrap() error { return e.Cause }

// NewError builds a domain error.
func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message}
}

// WrapError builds a domain error from an underlying cause.
func WrapError(code ErrorCode, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// NotFound returns a not-found domain error.
func NotFound(what, id string) *Error {
	return NewError(CodeNotFound, fmt.Sprintf("%s %q not found", what, id))
}

// InvalidInput returns an invalid-input domain error.
func InvalidInput(format string, args ...interface{}) *Error {
	return NewError(CodeInvalidInput, fmt.Sprintf(format, args...))
}

// Conflict returns a conflict domain error.
func Conflict(format string, args ...interface{}) *Error {
	return NewError(CodeConflict, fmt.Sprintf(format, args...))
}

// ErrorCodeOf extracts the ErrorCode from an error, defaulting to internal.
func ErrorCodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var de *Error
	if As(err, &de) {
		return de.Code
	}
	return CodeInternal
}

// As is a small helper mirroring errors.As for *Error without requiring
// callers to import the errors package in every layer.
func As(err error, target **Error) bool {
	for err != nil {
		if de, ok := err.(*Error); ok {
			*target = de
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
