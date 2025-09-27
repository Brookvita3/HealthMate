package user

import "errors"

var (
	// ErrUserNotFound is returned when a user is not found in the repository.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists is returned when trying to create a user with an email that already exists in the system.
	ErrUserAlreadyExists = errors.New("user with this email already exists")

	// ErrEmailAssociatedWithGoogle is returned when trying to register with an email that is already linked to a Google account.
	ErrEmailAssociatedWithGoogle = errors.New("this email is already associated with a Google account")

	// ErrPasswordNotSet is returned when a user created via Google tries to log in with a password before setting one.
	ErrPasswordNotSet = errors.New("this account was created with Google, please log in using your Google account or set a password")

	// ErrInvalidCredentials is a generic error for failed email/password login attempts.
	ErrInvalidCredentials = errors.New("invalid credentials")
)
