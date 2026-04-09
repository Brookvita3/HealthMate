package medication

import "errors"

var (
	ErrNotFound  = errors.New("medication or reminder not found")
	ErrForbidden = errors.New("medication does not belong to this user")
)
