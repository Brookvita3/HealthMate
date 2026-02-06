package validation

import (
	"auth-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

// MemberRepositoryAdapter wraps a MemberRepository to implement MemberChecker interface.
// This adapter pattern allows the validation package to work with member.MemberRepository
// without creating an import cycle.
type MemberRepositoryAdapter struct {
	repo MemberRepositoryInterface
}

// MemberRepositoryInterface defines what we need from MemberRepository
type MemberRepositoryInterface interface {
	GroupExists(ctx context.Context, groupID uuid.UUID) (bool, error)
	GetMember(ctx context.Context, groupID, userID uuid.UUID) (*domain.GroupMember, error)
	IsOwner(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
}

// NewMemberCheckerAdapter creates a new adapter for MemberRepository
func NewMemberCheckerAdapter(repo MemberRepositoryInterface) *MemberRepositoryAdapter {
	return &MemberRepositoryAdapter{repo: repo}
}

// GroupExists implements MemberChecker
func (a *MemberRepositoryAdapter) GroupExists(ctx context.Context, groupID uuid.UUID) (bool, error) {
	return a.repo.GroupExists(ctx, groupID)
}

// GetMember implements MemberChecker - converts *domain.GroupMember to interface{}
func (a *MemberRepositoryAdapter) GetMember(ctx context.Context, groupID, userID uuid.UUID) (interface{}, error) {
	return a.repo.GetMember(ctx, groupID, userID)
}

// IsOwner implements MemberChecker
func (a *MemberRepositoryAdapter) IsOwner(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	return a.repo.IsOwner(ctx, groupID, userID)
}
