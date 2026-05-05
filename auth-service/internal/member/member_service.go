package member

import (
	common "auth-service/internal/common"
	"auth-service/internal/domain"
	"auth-service/internal/user"
	"context"

	"github.com/google/uuid"
)

// Service defines the business logic for managing group members.
type Service interface {
	// InviteMember sends an invitation to a user to join a group.
	InviteMember(ctx context.Context, groupID, userID, invitedBy uuid.UUID) error

	// InviteByEmail sends an invitation to a user by their email.
	InviteByEmail(ctx context.Context, groupID uuid.UUID, email string, invitedBy uuid.UUID) error

	// AcceptInvitation allows a user to accept a group invitation.
	AcceptInvitation(ctx context.Context, groupID, userID uuid.UUID) error

	// RejectInvitation allows a user to reject a group invitation.
	RejectInvitation(ctx context.Context, groupID, userID uuid.UUID) error

	// UpdateMemberStatus updates the status of a membership.
	UpdateMemberStatus(ctx context.Context, groupID, userID uuid.UUID, status string) error

	// RemoveMember removes a user from a group (Kick functionality).
	RemoveMember(ctx context.Context, groupID, userID, requesterID uuid.UUID) error

	// LeaveGroup allows a user to leave a group.
	// If the user is the owner, they must transfer ownership first.
	LeaveGroup(ctx context.Context, groupID, userID uuid.UUID) error

	// GetMembers retrieves all members of a specific group.
	GetMembers(ctx context.Context, groupID uuid.UUID) ([]domain.GroupMember, error)

	// GetUserInvitations retrieves all pending invitations for a specific user.
	GetUserInvitations(ctx context.Context, userID uuid.UUID) ([]domain.InvitationResponse, error)

	// GetGroupInvitations retrieves pending invitations for a group with visibility logic.
	GetGroupInvitations(ctx context.Context, groupID, requesterID uuid.UUID) ([]domain.SentInvitationResponse, error)
}

type serviceImpl struct {
	memberRepo MemberRepository
	userRepo   user.UserRepository
}

func NewService(repo MemberRepository, userRepo user.UserRepository) Service {
	return &serviceImpl{
		memberRepo: repo,
		userRepo:   userRepo,
	}
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

// InviteByEmail implements Service.InviteByEmail.
func (s *serviceImpl) InviteByEmail(ctx context.Context, groupID uuid.UUID, email string, invitedBy uuid.UUID) error {
	// Find user by email
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return err // Should return ErrUserNotFound if not found
	}

	return s.InviteMember(ctx, groupID, user.Id, invitedBy)
}

// AcceptInvitation implements Service.AcceptInvitation.
// It changes the member status to 'accepted' and sets the joined time.
func (s *serviceImpl) AcceptInvitation(ctx context.Context, groupID, userID uuid.UUID) error {
	return s.UpdateMemberStatus(ctx, groupID, userID, "accepted")
}

// RejectInvitation implements Service.RejectInvitation.
// It changes the member status to 'rejected'.
func (s *serviceImpl) RejectInvitation(ctx context.Context, groupID, userID uuid.UUID) error {
	return s.UpdateMemberStatus(ctx, groupID, userID, "rejected")
}

// UpdateMemberStatus implements Service.UpdateMemberStatus.
func (s *serviceImpl) UpdateMemberStatus(ctx context.Context, groupID, userID uuid.UUID, status string) error {
	member, err := s.memberRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		return err
	}

	if member.Status == status {
		return nil
	}

	// Only members with 'pending' status can be updated to 'accepted' or 'rejected'
	if member.Status != "pending" {
		return ErrInvalidStatus
	}

	return s.memberRepo.UpdateMemberStatus(ctx, groupID, userID, status)
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
	// 1. Check if group exists
	exists, err := s.memberRepo.GroupExists(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrInvalidGroup
	}

	return s.memberRepo.ListGroupMembers(ctx, groupID)
}

// GetGroupInvitations implements Service.GetGroupInvitations.
func (s *serviceImpl) GetGroupInvitations(ctx context.Context, groupID, requesterID uuid.UUID) ([]domain.SentInvitationResponse, error) {
	// 1. Check if group exists
	exists, err := s.memberRepo.GroupExists(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrInvalidGroup
	}

	// 2. Check if requester is group owner or member
	isOwner, err := s.memberRepo.IsOwner(ctx, groupID, requesterID)
	if err != nil {
		return nil, err
	}

	requester, err := s.memberRepo.GetMember(ctx, groupID, requesterID)
	if err != nil {
		return nil, ErrNotMember
	}
	if requester.Status != "accepted" {
		return nil, ErrNotMember
	}

	invitations, err := s.memberRepo.GetGroupInvitations(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// 3. Filter invitations if the requester is not the owner
	if isOwner {
		return invitations, nil
	}

	filtered := []domain.SentInvitationResponse{}
	for _, inv := range invitations {
		if inv.InvitedBy == requesterID {
			filtered = append(filtered, inv)
		}
	}

	return filtered, nil
}

// GetUserInvitations implements Service.GetUserInvitations.
// Retrieves pending invitations for the user.
func (s *serviceImpl) GetUserInvitations(ctx context.Context, userID uuid.UUID) ([]domain.InvitationResponse, error) {
	invitations, err := s.memberRepo.GetUserInvitations(ctx, userID)
	if err != nil {
		return nil, err
	}

	for i := range invitations {
		if invitations[i].SharedMetrics == nil {
			invitations[i].SharedMetrics = []string{}
		}
	}

	return invitations, nil
}
