package user

import (
	"auth-service/internal/common"
)

var (
	// ErrUserNotFound is returned when a user is not found in the repository.
	ErrUserNotFound = &common.BusinessError{Code: 404, Message: "user not found"}

	// ErrUserAlreadyExists is returned when trying to create a user with an email that already exists in the system.
	ErrEmailAlreadyRegistered = &common.BusinessError{Code: 409, Message: "email is already registered"}

	// ErrPasswordNotSet is returned when a user created via Google tries to log in with a password before setting one.
	ErrPasswordNotSet = &common.BusinessError{Code: 400, Message: "password not set"}

	// ErrInvalidCredentials is a generic error for failed email/password login attempts.
	ErrInvalidCredentials = &common.BusinessError{Code: 401, Message: "invalid credentials"}
)
