package connection

import "errors"

var (
	// ErrConnectionNotFound is returned when a connection is not found in the repository.
	ErrConnectionNotFound = errors.New("connection not found")

	// ErrInvalidRequest is returned when the incoming request is invalid.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrConnectionExists is returned when a connection already exists in the repository.
	ErrConnectionExists = errors.New("connection already exists")

	// ErrInvalideStatus is returned when a connection has an invalid status.
	ErrInvalidStatus = errors.New("invalid status")
)
