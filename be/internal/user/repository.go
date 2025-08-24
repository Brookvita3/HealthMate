package user

import (
	"context"

	"github.com/google/uuid"
)

// ListUsersParams contains the parameters for filtering and paginating the user list.
type ListUsersParams struct {
	Search string // A search query to filter by name or email.
	Limit  int    // The number of users to return.
	Offset int    // The starting point for the user list.
	Status string // An optional filter for user status (e.g., 'active', 'banned').
}

// UpdateUserParams contains the parameters for updating a user's profile.
type UpdateUserParams struct {
	Name string
}

type Repository interface {
	// GetUserByEmail retrieves a single user by their unique email address.
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// GetUserByID retrieves a single user by their unique ID.
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)

	// CreateUser creates a new user in the database.
	CreateUser(ctx context.Context, user *User) error

	// UpdatePassword updates a user's password hash.
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error

	// UpdateUser updates a user's mutable profile data.
	UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUserParams) error

	// ListUsers retrieves a list of users based on filter and pagination parameters.
	ListUsers(ctx context.Context, params ListUsersParams) ([]User, error)
}
