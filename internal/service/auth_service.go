package service

import (
	"context"
	"errors"
	"log"

	"heathhub/internal/model"
	"heathhub/internal/repository"
	"heathhub/pkg/auth"

	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
)

type AuthResult struct {
	User         *model.User
	AccessToken  string
	RefreshToken string
	AccessTTL    int64
	RefreshTTL   int64
}

type gUser struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Locale        string `json:"locale"`
	Sub           string `json:"sub"`
}

type AuthService struct {
	UserRepo       *repository.UserRepository
	TokenService   *auth.TokenService
	googleClientID string
}

func NewAuthService(repo *repository.UserRepository, tokenService *auth.TokenService, googleClientID string) *AuthService {
	return &AuthService{UserRepo: repo, TokenService: tokenService, googleClientID: googleClientID}
}

func (s *AuthService) LoginWithGoogleIDToken(ctx context.Context, idToken string) (*AuthResult, error) {

	gUser, err := s.verifyGoogleIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	existing, err := s.UserRepo.FindByEmail(gUser.Email)
	if err != nil {
		return nil, err
	}

	var user *model.User
	if existing != nil {
		user = existing
	} else {
		newUser := &model.User{
			ID:       uuid.New(),
			Email:    gUser.Email,
			Name:     gUser.Name,
			Picture:  gUser.Picture,
			Provider: "google",
		}
		if err := s.UserRepo.Create(newUser); err != nil {
			return nil, err
		}
		user = newUser
	}

	accessToken, err := s.TokenService.GenerateAccessJWT(user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.TokenService.GenerateRefreshJWT(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil

}

func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {

	email, err := s.TokenService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	newAccessToken, err := s.TokenService.GenerateAccessJWT(email)
	if err != nil {
		log.Printf("error when generating new access token for  %s: %v", email, err)
		return "", errors.New("failed to generate new access token")
	}

	return newAccessToken, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {

	err := s.TokenService.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		log.Printf("error when revoking refresh token: %v", err)
		return errors.New("failed to revoke refresh token")
	}
	return nil
}

func (s *AuthService) verifyGoogleIDToken(ctx context.Context, idToken string) (*gUser, error) {

	payload, err := idtoken.Validate(ctx, idToken, s.googleClientID)
	if err != nil {
		log.Println("Lỗi xác thực ID token:", err)
		return nil, errors.New("invalid Google token")
	}

	gUser := &gUser{
		Email:         payload.Claims["email"].(string),
		EmailVerified: payload.Claims["email_verified"].(bool),
		Name:          payload.Claims["name"].(string),
		Picture:       payload.Claims["picture"].(string),
		GivenName:     payload.Claims["given_name"].(string),
		FamilyName:    payload.Claims["family_name"].(string),
		Sub:           payload.Claims["sub"].(string),
	}

	if payload.Issuer != "accounts.google.com" && payload.Issuer != "https://accounts.google.com" {
		return nil, errors.New("invalid token issuer")
	}

	return gUser, nil
}
