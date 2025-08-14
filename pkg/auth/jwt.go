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
	Rdb    *redis.Client
}

func NewTokenService(secret string, rdb *redis.Client) *TokenService {
	return &TokenService{
		secret: secret,
		Rdb:    rdb,
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

func (t *TokenService) GenerateRefreshJWT(email string) (string, error) {

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

	if err := t.Rdb.Set(context.Background(), "refresh:"+jti, email, exp_time).Err(); err != nil {
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
