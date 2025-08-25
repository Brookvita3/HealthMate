package admin

import (
	"context"

	"healthmate/internal/user"

	"github.com/google/uuid"
)

type Service interface {
	ListUsers(ctx context.Context, params user.ListUsersParams) ([]user.User, error)
	BanUser(ctx context.Context, id uuid.UUID) error
	UnbanUser(ctx context.Context, id uuid.UUID) error
}

type serviceImpl struct {
	userRepo user.Repository
}

func NewAdminService(userRepo user.Repository) Service {
	return &serviceImpl{userRepo: userRepo}
}

func (s *serviceImpl) ListUsers(ctx context.Context, params user.ListUsersParams) ([]user.User, error) {
	const maxLimit = 100
	if params.Limit > maxLimit {
		params.Limit = maxLimit
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	return s.userRepo.ListUsers(ctx, params)
}

func (s *serviceImpl) BanUser(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.UpdateStatus(ctx, id, "banned")
}

func (s *serviceImpl) UnbanUser(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.UpdateStatus(ctx, id, "active")
}
