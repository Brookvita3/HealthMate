package group

import (
	"auth-service/internal/common"
	"auth-service/internal/domain"
	"auth-service/internal/member"
	"auth-service/internal/user"
	"context"
	"errors"

	"github.com/google/uuid"
)

// Service defines the business logic for group operations
type Service interface {
	// CreateGroup creates a new group with the given owner
	CreateGroup(ctx context.Context, ownerID uuid.UUID, name string, description *string) (*domain.Group, error)

	// GetGroup retrieves a group by ID
	GetGroup(ctx context.Context, groupID uuid.UUID) (*domain.Group, error)

	// UpdateGroup updates an existing group
	UpdateGroup(ctx context.Context, groupID uuid.UUID, name, description *string, requesterID uuid.UUID) error

	// DeleteGroup deletes a group (soft delete)
	DeleteGroup(ctx context.Context, groupID, requesterID uuid.UUID) error

	// ListUserGroups lists all groups owned by a user
	ListUserGroups(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Group, error)

	// ListGroups lists groups with filters
	ListGroups(ctx context.Context, params ListGroupsParams) ([]domain.Group, error)

	// TransferOwnership transfers group ownership to another user
	TransferOwnership(ctx context.Context, groupID, currentOwnerID, newOwnerID uuid.UUID) error

	// IsGroupOwner checks if a user is the owner of a group
	IsGroupOwner(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
}

// serviceImpl implements the group.Service interface
type serviceImpl struct {
	groupRepo  GroupRepository
	userRepo   user.UserRepository
	memberRepo member.MemberRepository
}

// NewService creates a new group service instance
func NewService(groupRepo GroupRepository, userRepo user.UserRepository, memberRepo member.MemberRepository) Service {
	return &serviceImpl{
		groupRepo:  groupRepo,
		userRepo:   userRepo,
		memberRepo: memberRepo,
	}
}

// CreateGroup implements Service.CreateGroup
func (s *serviceImpl) CreateGroup(ctx context.Context, ownerID uuid.UUID, name string, description *string) (*domain.Group, error) {
	// Validate inputs
	if err := s.validateGroupName(name); err != nil {
		return nil, err
	}

	// Check if owner exists
	ownerExists, err := s.userRepo.Exists(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if !ownerExists {
		return nil, ErrNotGroupOwner
	}

	// Check if group name already exists for this owner
	if _, err := s.groupRepo.FindByNameAndOwner(ctx, name, ownerID); err == nil {
		return nil, ErrGroupAlreadyExists
	} else if !errors.Is(err, ErrGroupNotFound) {
		return nil, err
	}

	// Create group
	params := CreateGroupParams{
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
	}

	return s.groupRepo.Create(ctx, params)
}

// GetGroup implements Service.GetGroup
func (s *serviceImpl) GetGroup(ctx context.Context, groupID uuid.UUID) (*domain.Group, error) {
	return s.groupRepo.FindByID(ctx, groupID)
}

// UpdateGroup implements Service.UpdateGroup
func (s *serviceImpl) UpdateGroup(ctx context.Context, groupID uuid.UUID, name, description *string, requesterID uuid.UUID) error {
	// Check if requester is the group owner
	isOwner, err := s.IsGroupOwner(ctx, groupID, requesterID)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrNotGroupOwner
	}

	// Validate name if provided
	if name != nil {
		if err := s.validateGroupName(*name); err != nil {
			return err
		}
	}

	params := UpdateGroupParams{
		Name:        name,
		Description: description,
	}

	return s.groupRepo.Update(ctx, groupID, params)
}

// DeleteGroup implements Service.DeleteGroup
func (s *serviceImpl) DeleteGroup(ctx context.Context, groupID, requesterID uuid.UUID) error {
	// Check if requester is the group owner
	isOwner, err := s.IsGroupOwner(ctx, groupID, requesterID)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrNotGroupOwner
	}

	// Requirement: Owner must kick all users before deleting the group
	count, err := s.memberRepo.CountMembers(ctx, groupID)
	if err != nil {
		return err
	}
	if count > 1 {
		return &common.BusinessError{Code: 400, Message: "group must be empty except for the owner before deletion"}
	}

	return s.groupRepo.Delete(ctx, groupID)
}

// ListUserGroups implements Service.ListUserGroups
func (s *serviceImpl) ListUserGroups(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Group, error) {
	// Returns both owned and joined groups
	return s.groupRepo.FindByUser(ctx, userID, limit, offset)
}

// ListGroups implements Service.ListGroups
func (s *serviceImpl) ListGroups(ctx context.Context, params ListGroupsParams) ([]domain.Group, error) {
	return s.groupRepo.List(ctx, params)
}

// TransferOwnership implements Service.TransferOwnership
func (s *serviceImpl) TransferOwnership(ctx context.Context, groupID, currentOwnerID, newOwnerID uuid.UUID) error {
	// Verify current ownership
	isOwner, err := s.IsGroupOwner(ctx, groupID, currentOwnerID)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrNotGroupOwner
	}

	// Check if new owner exists
	newOwnerExists, err := s.userRepo.Exists(ctx, newOwnerID)
	if err != nil {
		return err
	}
	if !newOwnerExists {
		return ErrNotGroupOwner // Or a more specific error
	}

	// Cannot transfer to yourself (optional check)
	if currentOwnerID == newOwnerID {
		return nil // Or return an error if you want
	}

	return s.groupRepo.TransferOwnership(ctx, groupID, newOwnerID)
}

// IsGroupOwner implements Service.IsGroupOwner
func (s *serviceImpl) IsGroupOwner(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return false, err
	}
	return group.OwnerID == userID, nil
}

// validateGroupName validates the group name
func (s *serviceImpl) validateGroupName(name string) error {
	if len(name) < 3 {
		return ErrInvalidGroupName
	}
	if len(name) > 100 {
		return ErrInvalidGroupName
	}
	return nil
}
