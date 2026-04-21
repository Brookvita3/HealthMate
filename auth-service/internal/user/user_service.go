package user

import (
	"auth-service/internal/common"
	"auth-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

type Service interface {
	GetUserProfile(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateUserProfile(ctx context.Context, id uuid.UUID, params UpdateUserParams) error
	ListUsers(ctx context.Context, params ListUsersParams) ([]domain.User, error)
}

type serviceImpl struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) GetUserProfile(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.GetUserById(ctx, id)
}

func (s *serviceImpl) UpdateUserProfile(ctx context.Context, id uuid.UUID, params UpdateUserParams) error {
	if params.Name == nil && params.Picture == nil && params.Phone == nil && params.Address == nil &&
		params.Gender == nil && params.Birthday == nil && params.Weight == nil && params.Height == nil &&
		params.BloodGroup == nil {
		return common.ErrInvalidRequest
	}
	return s.repo.UpdateUser(ctx, id, params)
}

func (s *serviceImpl) ListUsers(ctx context.Context, params ListUsersParams) ([]domain.User, error) {
	const maxLimit = 100
	if params.Limit > maxLimit {
		params.Limit = maxLimit
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}

	return s.repo.ListUsers(ctx, params)
}
