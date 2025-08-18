package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		c.Set("userID", claims["sub"])
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

	accessToken, err := s.tokenService.GenerateAccessJWT(user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshJWT(ctx, user.Email)
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

	newAccessToken, err := s.tokenService.GenerateAccessJWT(email)
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
