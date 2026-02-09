package group

import (
	"auth-service/internal/common"
)

var (
	// ErrGroupNotFound is returned when a group is not found.
	ErrGroupNotFound = &common.BusinessError{Code: 404, Message: "group not found"}

	// ErrInvalidGroupName is returned when group name is invalid.
	ErrInvalidGroupName = &common.BusinessError{Code: 400, Message: "invalid group name"}

	// ErrNotGroupOwner is returned when user is not the group owner.
	ErrNotGroupOwner = &common.BusinessError{Code: 403, Message: "user is not the group owner"}

	// ErrGroupAlreadyExists is returned when trying to create a duplicate group.
	ErrGroupAlreadyExists = &common.BusinessError{Code: 409, Message: "group already exists"}

	// ErrNotMember is returned when the user is not a member of the group.
	ErrNotMember = &common.BusinessError{Code: 403, Message: "user is not a member of this group"}
)
