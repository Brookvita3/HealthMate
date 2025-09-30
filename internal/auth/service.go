package auth

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"healthmate/internal/common"
	"healthmate/internal/user"
)

type LoginResult struct {
	User         *user.User `json:"user"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
}

type gUser struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"`
}

type Service interface {
	RegisterWithEmail(ctx context.Context, email, password, name string) (*user.User, error)
	LoginWithGoogleIDToken(ctx context.Context, idToken string) (*LoginResult, error)
	LoginWithEmail(ctx context.Context, email, password string) (*LoginResult, error)
	SetPasswordForUser(ctx context.Context, id uuid.UUID, newPassword string) error
	RefreshAccessToken(ctx context.Context, refreshToken string) (string, error)
	Logout(ctx context.Context, refreshToken string) error
}

type serviceImpl struct {
	userRepo       user.Repository
	tokenService   TokenService
	googleClientID string
}

func NewAuthService(repo user.Repository, tokenService TokenService, googleClientID string) Service {
	return &serviceImpl{
		userRepo:       repo,
		tokenService:   tokenService,
		googleClientID: googleClientID,
	}
}

func (s *serviceImpl) RegisterWithEmail(ctx context.Context, email, password, name string) (*user.User, error) {
	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	if existing != nil {
		return nil, user.ErrEmailAlreadyRegistered
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return nil, common.ErrInternalServer
	}

	newUser := &user.User{
		Id:       uuid.New(),
		Email:    email,
		Name:     name,
		Provider: "HealthMate",
		Password: string(hashedPassword),
		Role:     "user",
		Status:   "unverified",
	}

	if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *serviceImpl) LoginWithEmail(ctx context.Context, email, password string) (*LoginResult, error) {
	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, user.ErrInvalidCredentials
	}

	if existing.Password == "" {
		return nil, user.ErrPasswordNotSet
	}

	err = bcrypt.CompareHashAndPassword([]byte(existing.Password), []byte(password))
	if err != nil {
		return nil, user.ErrInvalidCredentials
	}

	accessToken, err := s.tokenService.GenerateAccessToken(existing)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(ctx, existing)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         existing,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *serviceImpl) SetPasswordForUser(ctx context.Context, id uuid.UUID, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password for user %s: %v", id, err)
		return common.ErrInternalServer
	}
	return s.userRepo.UpdatePassword(ctx, id, string(hashedPassword))
}

func (s *serviceImpl) LoginWithGoogleIDToken(ctx context.Context, idToken string) (*LoginResult, error) {
	gUser, err := s.verifyGoogleIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	existing, err := s.userRepo.GetUserByEmail(ctx, gUser.Email)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	var userToAuth *user.User
	if existing != nil {
		userToAuth = existing
	} else {
		newUser := &user.User{
			Id:       uuid.New(),
			Email:    gUser.Email,
			Name:     gUser.Name,
			Provider: "Google",
			GoogleID: gUser.Sub,
			Picture:  gUser.Picture,
			Role:     "user",
			Status:   "active",
		}
		if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
			return nil, err
		}
		userToAuth = newUser
	}

	accessToken, err := s.tokenService.GenerateAccessToken(userToAuth)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(ctx, userToAuth)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         userToAuth,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *serviceImpl) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	userId, err := s.tokenService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	id, err := uuid.Parse(userId)
	if err != nil {
		return "", common.ErrInvalidUUIDFormat
	}

	user, err := s.userRepo.GetUserById(ctx, id)
	if err != nil {
		return "", err
	}

	return s.tokenService.GenerateAccessToken(user)
}

func (s *serviceImpl) Logout(ctx context.Context, refreshToken string) error {
	err := s.tokenService.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		log.Printf("error when revoking refresh token: %v", err)
		return err
	}
	return nil
}

func (s *serviceImpl) verifyGoogleIDToken(ctx context.Context, idToken string) (*gUser, error) {
	payload, err := idtoken.Validate(ctx, idToken, s.googleClientID)
	if err != nil {
		log.Println("error when verifying ID token:", err)
		return nil, common.ErrInvalidRequest
	}

	gUser := &gUser{
		Email:         payload.Claims["email"].(string),
		EmailVerified: payload.Claims["email_verified"].(bool),
		Name:          payload.Claims["name"].(string),
		Picture:       payload.Claims["picture"].(string),
		Sub:           payload.Claims["sub"].(string),
	}
	return gUser, nil
}
