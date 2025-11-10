package grpcserver

import (
	"auth-service/internal/auth"
	authpb "auth-service/proto/pb"
	"context"
)

type AuthGRPCServer struct {
	authpb.UnimplementedAuthServiceServer
	JwtService *auth.JWTTokenService
}

func (s *AuthGRPCServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	claims, err := s.JwtService.ValidateToken(req.Token)
	if err != nil {
		return &authpb.ValidateTokenResponse{Valid: false, Error: err.Error()}, nil
	}

	return &authpb.ValidateTokenResponse{
		Valid:  true,
		UserId: claims["sub"].(string),
		Email:  claims["email"].(string),
	}, nil
}
