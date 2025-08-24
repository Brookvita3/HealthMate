package user

import (
	"context"
	"github.com/google/uuid"
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	Create(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	// FindByGoogleID(ctx context.Context, googleID string) (*User, error)
	// UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	// UpdateProfile(ctx context.Context, id uuid.UUID, name string, picture string) error
}
