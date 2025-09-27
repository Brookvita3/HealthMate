package realtime

import "errors"

var (
	// ErrInvalidPayloadFormat is returned when a message payload cannot be unmarshalled or is malformed.
	ErrInvalidPayloadFormat = errors.New("invalid payload format")

	// ErrPermissionDenied is returned when a user attempts an action they are not authorized for.
	ErrPermissionDenied = errors.New("permission denied")
)
