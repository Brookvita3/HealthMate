package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"healthmate/pkg/jwtauth"
)

type LoginResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
}

type gUser struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"`
}

type serviceImpl struct {
	userRepo       Repository
	tokenService   *jwtauth.TokenService
	googleClientID string
}

func NewAuthService(repo Repository, tokenService *jwtauth.TokenService, googleClientID string) Service {
	return &serviceImpl{
		userRepo:       repo,
		tokenService:   tokenService,
		googleClientID: googleClientID,
	}
}

func (s *serviceImpl) RegisterWithEmail(ctx context.Context, email, password, name string) (*User, error) {
	existing, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if existing.Provider == "google" {
			return nil, errors.New("this email is already associated with a Google account. Please log in with Google and set a password in your account settings")
		}
		return nil, errors.New("user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return nil, errors.New("failed to process registration")
	}

	newUser := &User{
		ID:       uuid.New(),
		Email:    email,
		Name:     name,
		Provider: "HealthMate",
		Password: string(hashedPassword),
	}

	if err := s.userRepo.Create(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *serviceImpl) LoginWithEmail(ctx context.Context, email, password string) (*LoginResult, error) {
	existing, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, errors.New("invalid credentials")
	}

	if existing.Password == "" {
		return nil, errors.New("this account was created with Google, please log in using your Google account")
	}

	err = bcrypt.CompareHashAndPassword([]byte(existing.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	accessToken, err := s.tokenService.GenerateAccessJWT(existing.Email, existing.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshJWT(ctx, existing.Email, existing.ID.String())
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         existing,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *serviceImpl) SetPasswordForUser(ctx context.Context, email string, newPassword string) error {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password for user %s: %v", email, err)
		return errors.New("failed to set new password")
	}

	return s.userRepo.SetPassword(ctx, email, string(hashedPassword))
}

func (s *serviceImpl) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}

		claims, err := s.tokenService.ValidateJWT(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("email", claims["email"])
		c.Set("userId", claims["sub"])
		c.Next()
	}
}

func (s *serviceImpl) LoginWithGoogleIDToken(ctx context.Context, idToken string) (*LoginResult, error) {
	gUser, err := s.verifyGoogleIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	existing, err := s.userRepo.FindByEmail(gUser.Email)
	if err != nil {
		return nil, err
	}

	var user *User
	if existing != nil {
		user = existing
	} else {
		newUser := &User{
			ID:       uuid.New(),
			GoogleID: gUser.Sub,
			Email:    gUser.Email,
			Name:     gUser.Name,
			Picture:  gUser.Picture,
			Provider: "google",
		}
		if err := s.userRepo.Create(newUser); err != nil {
			return nil, err
		}
		user = newUser
	}

	accessToken, err := s.tokenService.GenerateAccessJWT(user.Email, user.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshJWT(ctx, user.Email, user.ID.String())
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *serviceImpl) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	email, err := s.tokenService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil || user == nil {
		return "", errors.New("user not found for refresh token")
	}

	newAccessToken, err := s.tokenService.GenerateAccessJWT(email, user.ID.String())
	if err != nil {
		log.Printf("error when generating new access token for %s: %v", email, err)
		return "", errors.New("failed to generate new access token")
	}

	return newAccessToken, nil
}

func (s *serviceImpl) Logout(ctx context.Context, refreshToken string) error {
	err := s.tokenService.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		log.Printf("error when revoking refresh token: %v", err)
		return errors.New("failed to revoke refresh token")
	}
	return nil
}

func (s *serviceImpl) verifyGoogleIDToken(ctx context.Context, idToken string) (*gUser, error) {
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
		Sub:           payload.Claims["sub"].(string),
	}
	return gUser, nil
}
