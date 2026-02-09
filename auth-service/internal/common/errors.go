package common

// BusinessError is a custom error type for business logic errors.
type BusinessError struct {
	Code    int
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

// Other errors
var (
	// ErrInvalidUUIDFormat is returned when a string value cannot be parsed into a valid UUID.
	ErrInvalidUUIDFormat = &BusinessError{Code: 400, Message: "invalid UUID format"}

	// ErrInvalidRequest is returned when the incoming request is invalid.
	ErrInvalidRequest = &BusinessError{Code: 400, Message: "invalid request"}

	// ErrInternalServer is returned when an unexpected internal server error occurs.
	ErrInternalServer = &BusinessError{Code: 500, Message: "internal server error"}

	// ErrMissingContextParam is returned when a required parameter in the context is missing.
	ErrMissingContextParam = &BusinessError{Code: 401, Message: "missing required parameter in context"}

	// ErrInvalidBody is returned when the request body is invalid or cannot be parsed.
	ErrInvalidBody = &BusinessError{Code: 400, Message: "invalid body"}
)
