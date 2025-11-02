package auth

import (
	"auth-service/internal/common"
	"auth-service/internal/domain"
	"context"
	"log"

	"cloud.google.com/go/auth/credentials/idtoken"
)

type GoogleTokenVerifier interface {
	VerifyGoogleIDToken(ctx context.Context, idToken string) (*domain.GoogleUser, error)
}

type GoogleTokenVerifierImpl struct {
	googleClientID string
}

func NewGoogleTokenVerifierImpl(googleClientID string) *GoogleTokenVerifierImpl {
	return &GoogleTokenVerifierImpl{
		googleClientID: googleClientID,
	}
}

func (g *GoogleTokenVerifierImpl) VerifyGoogleIDToken(ctx context.Context, idToken string) (*domain.GoogleUser, error) {
	payload, err := idtoken.Validate(ctx, idToken, g.googleClientID)
	if err != nil {
		log.Println("error when verifying ID token:", err)
		return nil, common.ErrInvalidRequest
	}

	gUser := &domain.GoogleUser{
		Email:         payload.Claims["email"].(string),
		EmailVerified: payload.Claims["email_verified"].(bool),
		Name:          payload.Claims["name"].(string),
		Picture:       payload.Claims["picture"].(string),
		Sub:           payload.Claims["sub"].(string),
	}
	return gUser, nil
}
