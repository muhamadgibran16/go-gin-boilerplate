// Package apperror defines application errors that carry a kind (not found, conflict, ...)
// and a message that is safe to show to API clients.
//
// Services return these errors; the HTTP layer (httpx.ErrorHandler) maps the kind to a
// status code. Any other error is treated as an internal error: it is logged and the
// client only sees a generic message.
//
//	var ErrNotFound = apperror.NotFound("user not found")
//	return nil, apperror.Internal("failed to save user", err) // wraps the cause for logs
package apperror

import (
	"errors"
	"net/http"
)

// Kind classifies an error
type Kind int

const (
	KindInternal Kind = iota
	KindBadRequest
	KindValidation
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindPayloadTooLarge
	KindTooManyRequests
	KindUnavailable
)

// HTTPStatus returns the HTTP status code for the kind
func (k Kind) HTTPStatus() int {
	switch k {
	case KindBadRequest, KindValidation:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindPayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case KindTooManyRequests:
		return http.StatusTooManyRequests
	case KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Error is an application error
type Error struct {
	Kind Kind
	// Message is shown to clients
	Message string
	// Fields holds per-field messages for validation errors
	Fields map[string]string
	// Err is the underlying cause. It is logged but never shown to clients.
	Err error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// Is makes errors.Is match two *Error with the same kind and message, so sentinel
// errors still match after being wrapped or recreated.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Kind == e.Kind && t.Message == e.Message
}

func newError(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

func BadRequest(message string) *Error      { return newError(KindBadRequest, message) }
func Unauthorized(message string) *Error    { return newError(KindUnauthorized, message) }
func Forbidden(message string) *Error       { return newError(KindForbidden, message) }
func NotFound(message string) *Error        { return newError(KindNotFound, message) }
func Conflict(message string) *Error        { return newError(KindConflict, message) }
func PayloadTooLarge(message string) *Error { return newError(KindPayloadTooLarge, message) }
func TooManyRequests(message string) *Error { return newError(KindTooManyRequests, message) }
func Unavailable(message string) *Error     { return newError(KindUnavailable, message) }

// Validation returns a validation error with per-field messages
func Validation(fields map[string]string) *Error {
	return &Error{Kind: KindValidation, Message: "Validation failed", Fields: fields}
}

// Internal returns an internal error wrapping cause. The message is logged, not shown.
func Internal(message string, cause error) *Error {
	return &Error{Kind: KindInternal, Message: message, Err: cause}
}

// Wrap returns a copy of e with cause attached, keeping its kind and message
func (e *Error) Wrap(cause error) *Error {
	c := *e
	c.Err = cause
	return &c
}

// From returns err as an *Error. Errors that are not application errors become internal errors.
func From(err error) *Error {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal("internal error", err)
}
