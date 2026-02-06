package member

import (
	common "auth-service/internal/common"
	"auth-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

// Service defines the business logic for managing group members.
type Service interface {
	// InviteMember sends an invitation to a user to join a group.
	InviteMember(ctx context.Context, groupID, userID, invitedBy uuid.UUID) error

	// AcceptInvitation allows a user to accept a group invitation.
	AcceptInvitation(ctx context.Context, groupID, userID uuid.UUID) error

	// RejectInvitation allows a user to reject a group invitation.
	RejectInvitation(ctx context.Context, groupID, userID uuid.UUID) error

	// RemoveMember removes a user from a group (Kick functionality).
	RemoveMember(ctx context.Context, groupID, userID, requesterID uuid.UUID) error

	// LeaveGroup allows a user to leave a group.
	// If the user is the owner, they must transfer ownership first.
	LeaveGroup(ctx context.Context, groupID, userID uuid.UUID) error

	// GetMembers retrieves all members of a specific group.
	GetMembers(ctx context.Context, groupID uuid.UUID) ([]domain.GroupMember, error)
}

type serviceImpl struct {
	memberRepo MemberRepository
}

func NewService(repo MemberRepository) Service {
	return &serviceImpl{memberRepo: repo}
}

// InviteMember implements Service.InviteMember.
// It checks if the user is already a member before adding them.
func (s *serviceImpl) InviteMember(ctx context.Context, groupID, userID, invitedBy uuid.UUID) error {
	// Check if group exists
	exists, err := s.memberRepo.GroupExists(ctx, groupID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrInvalidGroup
	}

	existingMem, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err == nil && existingMem != nil {
		return ErrMemberAlreadyExists
	}

	return s.memberRepo.AddMember(ctx, groupID, userID, invitedBy, "member")
}

// AcceptInvitation implements Service.AcceptInvitation.
// It changes the member status to 'accepted' and sets the joined time.
func (s *serviceImpl) AcceptInvitation(ctx context.Context, groupID, userID uuid.UUID) error {
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if member.Status == "accepted" {
		return nil
	}
	if member.Status != "pending" {
		return ErrInvalidStatus
	}
	return s.memberRepo.UpdateMemberStatus(ctx, groupID, userID, "accepted")
}

// RejectInvitation implements Service.RejectInvitation.
// It changes the member status to 'rejected'.
func (s *serviceImpl) RejectInvitation(ctx context.Context, groupID, userID uuid.UUID) error {
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if member.Status != "pending" {
		return ErrInvitationPending
	}
	return s.memberRepo.UpdateMemberStatus(ctx, groupID, userID, "rejected")
}

// RemoveMember implements Service.RemoveMember (Kick).
// It ensures that only the group owner can remove other members.
func (s *serviceImpl) RemoveMember(ctx context.Context, groupID, userID, requesterID uuid.UUID) error {

	// Check if group exists
	exists, err := s.memberRepo.GroupExists(ctx, groupID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrInvalidGroup
	}

	// Check if requester is group owner
	isOwner, err := s.memberRepo.IsOwner(ctx, groupID, requesterID)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrNotGroupOwner
	}

	// Owner cannot kick themselves
	if userID == requesterID {
		return &common.BusinessError{Code: 400, Message: "owner cannot kick themselves, use leave instead"}
	}

	return s.memberRepo.RemoveMember(ctx, groupID, userID)
}

// LeaveGroup implements Service.LeaveGroup.
// It checks if the user is the owner before allowing them to leave.
func (s *serviceImpl) LeaveGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	
	// Check if group exists
	exists, err := s.memberRepo.GroupExists(ctx, groupID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrInvalidGroup
	}
	
	// Check if user is the owner
	isOwner, err := s.memberRepo.IsOwner(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if isOwner {
		return ErrOwnerCannotLeave
	}

	return s.memberRepo.RemoveMember(ctx, groupID, userID)
}

// GetMembers implements Service.GetMembers.
// Lists all members in the group including their roles and statuses.
func (s *serviceImpl) GetMembers(ctx context.Context, groupID uuid.UUID) ([]domain.GroupMember, error) {
	return s.memberRepo.ListGroupMembers(ctx, groupID)
}
