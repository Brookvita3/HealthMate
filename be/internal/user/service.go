package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Service interface {
	GetUserProfile(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateUserProfile(ctx context.Context, id uuid.UUID, params UpdateUserParams) error
	ListUsers(ctx context.Context, params ListUsersParams) ([]User, error)
}

type serviceImpl struct {
	repo Repository
}

func NewUserService(repo Repository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) GetUserProfile(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *serviceImpl) UpdateUserProfile(ctx context.Context, id uuid.UUID, params UpdateUserParams) error {
	if params.Name == "" {
		return errors.New("user name cannot be empty")
	}
	return s.repo.UpdateUser(ctx, id, params)
}

func (s *serviceImpl) ListUsers(ctx context.Context, params ListUsersParams) ([]User, error) {
	const maxLimit = 100
	if params.Limit > maxLimit {
		params.Limit = maxLimit
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}

	return s.repo.ListUsers(ctx, params)
}
