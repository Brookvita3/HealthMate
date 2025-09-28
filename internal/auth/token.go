package auth

import (
	"context"
	"healthmate/internal/user"
)

type TokenService interface {
	GenerateAccessToken(user *user.User) (string, error)
	GenerateRefreshToken(ctx context.Context, user *user.User) (string, error)
	ValidateToken(tokenString string) (map[string]any, error)
	ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}
