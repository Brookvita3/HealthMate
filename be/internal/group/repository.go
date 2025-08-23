package group

import "context"

type Repository interface {
	CreateGroup(ctx context.Context, group *Group) error
	FindGroupByID(ctx context.Context, groupID string) (*Group, error)
	AddMember(ctx context.Context, member *GroupMember) error
	RemoveMember(ctx context.Context, groupID, userID string) error
	FindMembersByGroupID(ctx context.Context, groupID string) ([]*GroupMember, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
}
