package member

import (
	"auth-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

type MemberRepository interface {
	// AddMember inserts a new member into a group with a 'pending' status.
	// If the user is already a member, it does nothing.
	AddMember(ctx context.Context, groupID, userID, invitedBy uuid.UUID, role string) error

	// UpdateMemberStatus updates the status (e.g., 'accepted', 'rejected') of a group member.
	// If status is 'accepted', it also sets the joined_at timestamp.
	UpdateMemberStatus(ctx context.Context, groupID, userID uuid.UUID, status string) error

	// RemoveMember deletes a member from a group.
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error

	// GetMember retrieves detailed information about a specific group member.
	// Returns ErrMemberNotFound if the member does not exist in the group.
	GetMember(ctx context.Context, groupID, userID uuid.UUID) (*domain.GroupMember, error)

	// ListGroupMembers retrieves all members belonging to a specific group.
	ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]domain.GroupMember, error)

	// GroupExists checks if a group exists.
	GroupExists(ctx context.Context, groupID uuid.UUID) (bool, error)
}
