package auth

import (
	"auth-service/internal/cache"
	"auth-service/internal/domain"
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTTokenService struct {
	secret string
	cache  cache.Cache
}

func NewJWTTokenService(secret string, cache cache.Cache) *JWTTokenService {
	return &JWTTokenService{
		secret: secret,
		cache:  cache,
	}
}

func (t *JWTTokenService) GenerateAccessToken(user *domain.User) (string, error) {

	exp_time := 5 * time.Minute

	claims := jwt.MapClaims{
		"email": user.Email,
		"sub":   user.Id,
		"type":  "access",
		"role":  user.Role,
		"exp":   time.Now().Add(exp_time).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(t.secret))
}

func (t *JWTTokenService) GenerateRefreshToken(ctx context.Context, user *domain.User) (string, error) {

	jti := uuid.New().String()
	exp_time := 1 * time.Hour

	claims := jwt.MapClaims{
		"email": user.Email,
		"sub":   user.Id,
		"type":  "refresh",
		"role":  user.Role,
		"exp":   time.Now().Add(exp_time).Unix(),
		"iat":   time.Now().Unix(),
		"jti":   jti,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(t.secret))
	if err != nil {
		return "", err
	}

	if err := t.cache.Set(ctx, "refresh:"+jti, user.Id, exp_time); err != nil {
		return "", err
	}

	return signedToken, nil
}

func (t *JWTTokenService) ValidateToken(tokenString string) (map[string]any, error) {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(t.secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Return userId from refresh token if valid
func (t *JWTTokenService) ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error) {

	claims, err := t.ValidateToken(refreshToken)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	if claims["type"] != "refresh" {
		return "", errors.New("token is not refresh type")
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return "", errors.New("missing jti")
	}

	_, ok = claims["email"].(string)
	if !ok {
		return "", errors.New("missing email")
	}

	val, err := t.cache.Get(ctx, "refresh:"+jti)
	if err != nil {
		return "", errors.New("refresh token expired or revoked")
	}

	return val, nil
}

func (t *JWTTokenService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {

	claims, err := t.ValidateToken(refreshToken)
	if err != nil {
		return errors.New("invalid refresh token")
	}

	if claims["type"] != "refresh" {
		return errors.New("token is not refresh type")
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return errors.New("missing jti")
	}

	return t.cache.Delete(ctx, "refresh:"+jti)
}
