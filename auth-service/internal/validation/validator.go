package validation

import (
	"auth-service/internal/common"
	"auth-service/internal/group"
	"auth-service/internal/member"
	"context"

	"github.com/google/uuid"
)

// ErrUserNotFound is returned when a user is not found.
// This error is unique to validation package as it's a cross-cutting concern.
var ErrUserNotFound = &common.BusinessError{Code: 404, Message: "user not found"}

// GroupChecker defines the interface for group-related checks
type GroupChecker interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// UserChecker defines the interface for user-related checks
type UserChecker interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// MemberChecker defines the interface for member-related checks
type MemberChecker interface {
	GroupExists(ctx context.Context, groupID uuid.UUID) (bool, error)
	GetMember(ctx context.Context, groupID, userID uuid.UUID) (interface{}, error)
	IsOwner(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
}

// Validator provides centralized validation logic for common entities
type Validator struct {
	groupChecker  GroupChecker
	userChecker   UserChecker
	memberChecker MemberChecker
}

// NewValidator creates a new Validator instance
func NewValidator(groupChecker GroupChecker, userChecker UserChecker, memberChecker MemberChecker) *Validator {
	return &Validator{
		groupChecker:  groupChecker,
		userChecker:   userChecker,
		memberChecker: memberChecker,
	}
}

// ValidateGroupExists checks if a group exists
func (v *Validator) ValidateGroupExists(ctx context.Context, groupID uuid.UUID) error {
	exists, err := v.groupChecker.Exists(ctx, groupID)
	if err != nil {
		return err
	}
	if !exists {
		return group.ErrGroupNotFound
	}
	return nil
}

// ValidateUserExists checks if a user exists
func (v *Validator) ValidateUserExists(ctx context.Context, userID uuid.UUID) error {
	exists, err := v.userChecker.Exists(ctx, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	return nil
}

// ValidateMemberExists checks if a user is a member of a group
func (v *Validator) ValidateMemberExists(ctx context.Context, groupID, userID uuid.UUID) error {
	m, err := v.memberChecker.GetMember(ctx, groupID, userID)
	if err != nil {
		return member.ErrMemberNotFound
	}
	if m == nil {
		return member.ErrMemberNotFound
	}
	return nil
}

// ValidateIsGroupOwner checks if a user is the owner of a group
func (v *Validator) ValidateIsGroupOwner(ctx context.Context, groupID, userID uuid.UUID) error {
	isOwner, err := v.memberChecker.IsOwner(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return group.ErrNotGroupOwner
	}
	return nil
}

// ValidateGroupAndUser validates both group and user exist
func (v *Validator) ValidateGroupAndUser(ctx context.Context, groupID, userID uuid.UUID) error {
	if err := v.ValidateGroupExists(ctx, groupID); err != nil {
		return err
	}
	if err := v.ValidateUserExists(ctx, userID); err != nil {
		return err
	}
	return nil
}

// ValidateGroupAndMember validates group exists and user is a member
func (v *Validator) ValidateGroupAndMember(ctx context.Context, groupID, userID uuid.UUID) error {
	if err := v.ValidateGroupExists(ctx, groupID); err != nil {
		return err
	}
	if err := v.ValidateMemberExists(ctx, groupID, userID); err != nil {
		return err
	}
	return nil
}

// ValidateGroupOwnership validates group exists and user is the owner
func (v *Validator) ValidateGroupOwnership(ctx context.Context, groupID, userID uuid.UUID) error {
	if err := v.ValidateGroupExists(ctx, groupID); err != nil {
		return err
	}
	if err := v.ValidateIsGroupOwner(ctx, groupID, userID); err != nil {
		return err
	}
	return nil
}
