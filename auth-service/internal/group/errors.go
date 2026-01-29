package group

import "errors"

var (
	// ErrGroupNotFound is returned when a group is not found
	ErrGroupNotFound = errors.New("group not found")

	// ErrInvalidGroupName is returned when group name is invalid
	ErrInvalidGroupName = errors.New("invalid group name")

	// ErrNotGroupOwner is returned when user is not the group owner
	ErrNotGroupOwner = errors.New("user is not the group owner")

	// ErrGroupAlreadyExists is returned when trying to create a duplicate group
	ErrGroupAlreadyExists = errors.New("group already exists")
)
