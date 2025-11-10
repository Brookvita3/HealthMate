package authclient

import (
	authpb "api-gateway/proto/pb"
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthGRPCClient struct {
	client authpb.AuthServiceClient
}

func NewAuthGRPCClient(addr string) (*AuthGRPCClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &AuthGRPCClient{
		client: authpb.NewAuthServiceClient(conn),
	}, nil
}

func (a *AuthGRPCClient) ValidateToken(ctx context.Context, token string) (bool, string, string, error) {
	res, err := a.client.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: token})
	if err != nil {
		return false, "", "", err
	}

	if !res.Valid {
		return false, "", "", fmt.Errorf("Error : %s", res.Error)
	}

	return true, res.UserId, res.Email, nil
}
