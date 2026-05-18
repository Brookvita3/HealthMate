package user

import (
	"auth-service/internal/common"
	"auth-service/internal/domain"
	"context"
	"strings"

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
		params.BloodGroup == nil && params.Timezone == nil && params.Allergies == nil {
		return common.ErrInvalidRequest
	}

	if params.Allergies != nil {
		cleaned, err := validateAllergies(*params.Allergies)
		if err != nil {
			return err
		}
		params.Allergies = &cleaned
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

// validateAllergies trims, deduplicates (case-insensitive), and enforces limits.
func validateAllergies(raw []string) ([]string, error) {
	const maxItems = 50
	const maxItemLen = 100

	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, a := range raw {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if len([]rune(a)) > maxItemLen {
			a = string([]rune(a)[:maxItemLen])
		}
		lower := strings.ToLower(a)
		if !seen[lower] {
			seen[lower] = true
			out = append(out, a)
		}
	}
	if len(out) > maxItems {
		return nil, common.ErrInvalidRequest
	}
	return out, nil
}
