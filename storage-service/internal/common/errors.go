package common

// BusinessError is a custom error type for business logic errors.
type BusinessError struct {
	Code    int
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

var (
	ErrInvalidRequest = &BusinessError{Code: 400, Message: "invalid request"}
	ErrInternalServer = &BusinessError{Code: 500, Message: "internal server error"}
)
