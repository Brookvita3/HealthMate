package group

import (
	"context"
)

type Service interface {
	CheckMembership(ctx context.Context, userID, groupID string) (bool, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CheckMembership(ctx context.Context, userID, groupID string) (bool, error) {
	// Có thể thêm các logic khác ở đây nếu cần, ví dụ kiểm tra user có bị ban không...
	// return s.repo.IsUserMemberOfGroup(ctx, userID, groupID)
	return true, nil
}
