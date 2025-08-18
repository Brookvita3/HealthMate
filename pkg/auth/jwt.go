package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type TokenService struct {
	secret string
	rdb    *redis.Client
}

func NewTokenService(secret string, rdb *redis.Client) *TokenService {
	return &TokenService{
		secret: secret,
		rdb:    rdb,
	}
}

func (t *TokenService) GenerateAccessJWT(email string) (string, error) {

	exp_time := 5 * time.Minute

	claims := jwt.MapClaims{
		"email": email,
		"type":  "access",
		"exp":   time.Now().Add(exp_time).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(t.secret))
}

func (t *TokenService) GenerateRefreshJWT(ctx context.Context, email string) (string, error) {

	jti := uuid.New().String()
	exp_time := 1 * time.Hour

	claims := jwt.MapClaims{
		"email": email,
		"type":  "refresh",
		"exp":   time.Now().Add(exp_time).Unix(),
		"iat":   time.Now().Unix(),
		"jti":   jti,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(t.secret))
	if err != nil {
		return "", err
	}

	if err := t.rdb.Set(ctx, "refresh:"+jti, email, exp_time).Err(); err != nil {
		return "", err
	}

	return signedToken, nil
}

func (t *TokenService) ValidateJWT(tokenString string) (jwt.MapClaims, error) {

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

func (t *TokenService) ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error) {

	claims, err := t.ValidateJWT(refreshToken)
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

	email, ok := claims["email"].(string)
	if !ok {
		return "", errors.New("missing email")
	}

	val, err := t.rdb.Get(ctx, "refresh:"+jti).Result()
	if err != nil || val != email {
		return "", errors.New("refresh token expired or revoked")
	}

	return email, nil
}

func (t *TokenService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {

	claims, err := t.ValidateJWT(refreshToken)
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

	return t.rdb.Del(ctx, "refresh:"+jti).Err()
}
