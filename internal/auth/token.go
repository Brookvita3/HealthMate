package auth

import "context"

type TokenService interface {
	GenerateAccessToken(email, id, role string) (string, error)
	GenerateRefreshToken(ctx context.Context, email, id, role string) (string, error)
	ValidateToken(tokenString string) (map[string]any, error)
	ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}
