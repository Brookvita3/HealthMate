package member

import (
	"auth-service/internal/common"
)

var (
	// ErrMemberNotFound is returned when a member is not found in the group.
	ErrMemberNotFound = &common.BusinessError{Code: 404, Message: "member not found in group"}

	// ErrMemberAlreadyExists is returned when a user is already a member of the group.
	ErrMemberAlreadyExists = &common.BusinessError{Code: 409, Message: "user is already a member of this group"}

	// ErrNotMember is returned when the user is not a member of the group.
	ErrNotMember = &common.BusinessError{Code: 403, Message: "user is not a member of this group"}

	// ErrInvitationPending is returned when an invitation is still pending.
	ErrInvitationPending = &common.BusinessError{Code: 400, Message: "invitation is still pending"}

	// ErrInvalidStatus is returned when the status is invalid.
	ErrInvalidStatus = &common.BusinessError{Code: 400, Message: "invalid status"}

	// ErrInvalidGroup is returned when the request is invalid.
	ErrInvalidGroup = &common.BusinessError{Code: 404, Message: "invalid group"}

	// ErrOwnerCannotLeave is returned when the owner attempts to leave the group.
	ErrOwnerCannotLeave = &common.BusinessError{Code: 400, Message: "owner must transfer leadership before leaving"}

	// ErrNotGroupOwner is returned when a non-owner attempts an owner-only action.
	ErrNotGroupOwner = &common.BusinessError{Code: 403, Message: "user is not the group owner"}
)
