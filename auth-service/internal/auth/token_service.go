package auth

import (
	"auth-service/internal/domain"
	"context"
)

type TokenService interface {
	GenerateAccessToken(user *domain.User) (string, error)
	GenerateRefreshToken(ctx context.Context, user *domain.User) (string, error)
	ValidateToken(tokenString string) (map[string]any, error)
	ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}
