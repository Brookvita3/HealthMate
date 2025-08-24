package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"healthmate/internal/common"
	"healthmate/internal/user"
	"healthmate/pkg/jwtauth"
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

type serviceImpl struct {
	userRepo       user.Repository
	tokenService   *jwtauth.TokenService
	googleClientID string
}

func NewAuthService(repo user.Repository, tokenService *jwtauth.TokenService, googleClientID string) Service { // << CHANGED
	return &serviceImpl{
		userRepo:       repo,
		tokenService:   tokenService,
		googleClientID: googleClientID,
	}
}

func (s *serviceImpl) RegisterWithEmail(ctx context.Context, email, password, name string) (*user.User, error) { // << CHANGED return type
	existing, err := s.userRepo.GetUserByEmail(ctx, email)
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

	newUser := &user.User{
		ID:       uuid.New(),
		Email:    email,
		Name:     name,
		Provider: "HealthMate",
		Password: pgtype.Text{String: string(hashedPassword), Valid: true},
		Role:     "user",
		Status:   "active",
	}

	if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *serviceImpl) LoginWithEmail(ctx context.Context, email, password string) (*LoginResult, error) {
	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, errors.New("invalid credentials")
	}

	if !existing.Password.Valid {
		return nil, errors.New("this account was created with Google, please log in using your Google account")
	}

	err = bcrypt.CompareHashAndPassword([]byte(existing.Password.String), []byte(password))
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

func (s *serviceImpl) SetPasswordForUser(ctx context.Context, id string, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password for user %s: %v", id, err)
		return errors.New("failed to set new password")
	}

	userId, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid user ID format")
	}
	return s.userRepo.UpdatePassword(ctx, userId, string(hashedPassword))
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

		c.Set(string(common.EmailKey), claims["email"])
		c.Set(string(common.UserIdKey), claims["sub"])
		c.Next()
	}
}

func (s *serviceImpl) LoginWithGoogleIDToken(ctx context.Context, idToken string) (*LoginResult, error) {
	gUser, err := s.verifyGoogleIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	existing, err := s.userRepo.GetUserByEmail(ctx, gUser.Email)
	if err != nil {
		return nil, err
	}

	var userToAuth *user.User
	if existing != nil {
		userToAuth = existing
	} else {
		newUser := &user.User{
			ID:       uuid.New(),
			Email:    gUser.Email,
			Name:     gUser.Name,
			Provider: "google",
			GoogleID: pgtype.Text{String: gUser.Sub, Valid: true},
			Picture:  pgtype.Text{String: gUser.Picture, Valid: true},
			Password: pgtype.Text{Valid: false},
			Role:     "user",
			Status:   "active",
		}
		if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
			return nil, err
		}
		userToAuth = newUser
	}

	accessToken, err := s.tokenService.GenerateAccessJWT(userToAuth.Email, userToAuth.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshJWT(ctx, userToAuth.Email, userToAuth.ID.String())
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
	userIdStr, err := s.tokenService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		return "", errors.New("invalid user ID format in refresh token")
	}

	user, err := s.userRepo.GetUserByID(ctx, userId)
	if err != nil || user == nil {
		return "", errors.New("user not found for refresh token")
	}

	newAccessToken, err := s.tokenService.GenerateAccessJWT(user.Email, user.ID.String())
	if err != nil {
		log.Printf("error when generating new access token for %s: %v", user.Email, err)
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
		log.Println("error when verifying ID token:", err)
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
