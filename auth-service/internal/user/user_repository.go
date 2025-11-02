package user

import (
	"context"

	"auth-service/internal/domain"
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
	Name    *string
	Role    *string
	Status  *string
	Picture *string
}

type UserRepository interface {
	// GetUserByEmail searches for a user by email.
	// It returns ErrUserNotFound if no user is found.
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetUserByID searches for a user by id (UUID).
	// It returns ErrUserNotFound if no user is found.
	GetUserById(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// CreateUser inserts a new user record into the database.
	CreateUser(ctx context.Context, user *domain.User) error

	// UpdatePassword updates the hashed password for the user with the given email.
	// It returns ErrUserNotFound if no user is found.
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error

	// UpdateUser updates a user's mutable profile data (e.g., name).
	// It returns ErrUserNotFound if no user is found.
	UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUserParams) error

	// ListUsers retrieves a list of users based on filter and pagination parameters.
	// It dynamically and safely constructs the query to handle optional filters.
	ListUsers(ctx context.Context, params ListUsersParams) ([]domain.User, error)

	// UpdateStatus changes the status of a user.
	// It returns ErrUserNotFound if no user is found.
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}
