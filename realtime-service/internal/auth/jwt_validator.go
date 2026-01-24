package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// TokenValidator defines the interface for token validation.
type TokenValidator interface {
	ValidateToken(tokenString string) (*TokenClaims, error)
}

// TokenClaims contains the extracted claims from a validated token.
type TokenClaims struct {
	UserID string
	Email  string
	Role   string
}

// JWTValidator validates JWT tokens using a shared secret.
type JWTValidator struct {
	secret string
}

// NewJWTValidator creates a new JWTValidator with the given secret.
func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{secret: secret}
}

// ValidateToken parses and validates a JWT token string.
// Returns the extracted claims if valid, or an error if validation fails.
func (v *JWTValidator) ValidateToken(tokenString string) (*TokenClaims, error) {
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(v.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	userID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)

	if userID == "" {
		return nil, errors.New("missing user id in token")
	}

	return &TokenClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
	}, nil
}
