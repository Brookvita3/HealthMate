package common

import "errors"

var (
	// ErrInvalidUUIDFormat is returned when a string value cannot be parsed into a valid UUID. This usually indicates a malformed URL parameter or request body.
	ErrInvalidUUIDFormat = errors.New("invalid UUID format")

	// ErrInvalidRequest is returned when the incoming request is invalid.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrInternalServer is returned when an unexpected internal server error occurs.
	ErrInternalServer = errors.New("internal server error")

	// ErrMissingContextParam is returned when a required parameter in the context (userId, email, role) is missing.
	ErrMissingContextParam = errors.New("missing required parameter in context")

	// ErrInvalidBody is returned when the request body is invalid or cannot be parsed.
	ErrInvalidBody = errors.New("invalid body")
)
