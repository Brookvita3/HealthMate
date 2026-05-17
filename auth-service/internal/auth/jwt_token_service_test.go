package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"auth-service/internal/domain"
	"auth-service/mocks"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

type jwtTestEnv struct {
	cache   *mocks.Cache
	service *JWTTokenService
}

func newJWTTestEnv() *jwtTestEnv {
	c := new(mocks.Cache)
	return &jwtTestEnv{
		cache:   c,
		service: NewJWTTokenService("test-secret", c),
	}
}

func sampleUser() *domain.User {
	return &domain.User{
		Id:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Email: "user@example.com",
		Role:  "user",
	}
}

// signedToken creates a JWT with the given claims signed by the given key.
func signedToken(claims jwt.MapClaims, key string) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString([]byte(key))
	return s
}

// ─── GenerateAccessToken ─────────────────────────────────────────────────────

func TestGenerateAccessToken(t *testing.T) {
	t.Run("success: token contains correct claims and no cache interaction", func(t *testing.T) {
		// Access tokens are stateless — no cache call should be made.
		// Claims must include sub, email, type="access", role, and exp ~5 min ahead.
		env := newJWTTestEnv()
		user := sampleUser()

		tokenStr, err := env.service.GenerateAccessToken(user)

		assert.NoError(t, err)
		assert.NotEmpty(t, tokenStr)

		parsed := jwt.MapClaims{}
		tok, parseErr := jwt.ParseWithClaims(tokenStr, parsed, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		assert.NoError(t, parseErr)
		assert.True(t, tok.Valid)
		assert.Equal(t, user.Email, parsed["email"])
		assert.Equal(t, user.Id.String(), parsed["sub"])
		assert.Equal(t, "access", parsed["type"])
		assert.Equal(t, user.Role, parsed["role"])
		exp := time.Unix(int64(parsed["exp"].(float64)), 0)
		assert.WithinDuration(t, time.Now().Add(5*time.Minute), exp, 5*time.Second)
		env.cache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// ─── GenerateRefreshToken ─────────────────────────────────────────────────────

func TestGenerateRefreshToken(t *testing.T) {
	t.Run("success: token is stored in cache with correct key and value", func(t *testing.T) {
		// cache.Set must be called exactly once with key "refresh:<jti>" and
		// the user's UUID string as the value, with a 1-hour TTL.
		env := newJWTTestEnv()
		user := sampleUser()

		env.cache.On("Set",
			mock.Anything,
			mock.MatchedBy(func(k string) bool { return strings.HasPrefix(k, "refresh:") }),
			user.Id.String(),
			time.Hour,
		).Return(nil).Once()

		tokenStr, err := env.service.GenerateRefreshToken(context.Background(), user)

		assert.NoError(t, err)
		assert.NotEmpty(t, tokenStr)

		parsed := jwt.MapClaims{}
		tok, parseErr := jwt.ParseWithClaims(tokenStr, parsed, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		assert.NoError(t, parseErr)
		assert.True(t, tok.Valid)
		assert.Equal(t, "refresh", parsed["type"])
		assert.NotEmpty(t, parsed["jti"])
		exp := time.Unix(int64(parsed["exp"].(float64)), 0)
		assert.WithinDuration(t, time.Now().Add(time.Hour), exp, 5*time.Second)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: cache failure is returned immediately", func(t *testing.T) {
		// If cache.Set fails the error must propagate to the caller.
		env := newJWTTestEnv()
		user := sampleUser()
		cacheErr := errors.New("redis unavailable")

		env.cache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(cacheErr).Once()

		_, err := env.service.GenerateRefreshToken(context.Background(), user)

		assert.Error(t, err)
		env.cache.AssertExpectations(t)
	})
}

// ─── ValidateToken ───────────────────────────────────────────────────────────

func TestValidateToken(t *testing.T) {
	t.Run("success: valid access token returns claims map", func(t *testing.T) {
		// A freshly generated access token must pass validation without error.
		env := newJWTTestEnv()
		user := sampleUser()

		tokenStr, _ := env.service.GenerateAccessToken(user)
		claims, err := env.service.ValidateToken(tokenStr)

		assert.NoError(t, err)
		assert.Equal(t, "access", claims["type"])
		assert.Equal(t, user.Id.String(), claims["sub"])
	})

	t.Run("error: expired token is rejected", func(t *testing.T) {
		// A token whose exp is in the past must cause ValidateToken to error.
		env := newJWTTestEnv()
		tokenStr := signedToken(jwt.MapClaims{
			"sub":  "user-1",
			"type": "access",
			"exp":  time.Now().Add(-time.Hour).Unix(),
		}, "test-secret")

		claims, err := env.service.ValidateToken(tokenStr)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("error: token signed with wrong secret is rejected", func(t *testing.T) {
		// An HMAC signature produced with a different key must fail verification.
		env := newJWTTestEnv()
		tokenStr := signedToken(jwt.MapClaims{
			"sub":  "user-1",
			"type": "access",
			"exp":  time.Now().Add(time.Hour).Unix(),
		}, "wrong-secret")

		claims, err := env.service.ValidateToken(tokenStr)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("error: malformed token string is rejected", func(t *testing.T) {
		// Garbage input must return an error and must not panic.
		env := newJWTTestEnv()

		claims, err := env.service.ValidateToken("not.a.valid.jwt")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

// ─── ValidateRefreshToken ────────────────────────────────────────────────────

func TestValidateRefreshToken(t *testing.T) {
	t.Run("success: valid refresh token returns user ID stored in cache", func(t *testing.T) {
		// A token present in cache must return the stored user-ID value.
		env := newJWTTestEnv()
		user := sampleUser()
		jti := uuid.New().String()

		tokenStr := signedToken(jwt.MapClaims{
			"sub":   user.Id.String(),
			"email": user.Email,
			"type":  "refresh",
			"jti":   jti,
			"exp":   time.Now().Add(time.Hour).Unix(),
		}, "test-secret")

		env.cache.On("Get", mock.Anything, "refresh:"+jti).Return(user.Id.String(), nil).Once()

		got, err := env.service.ValidateRefreshToken(context.Background(), tokenStr)

		assert.NoError(t, err)
		assert.Equal(t, user.Id.String(), got)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: access token is rejected as wrong type", func(t *testing.T) {
		// Passing an access token must fail with "not refresh type" before cache lookup.
		env := newJWTTestEnv()
		user := sampleUser()

		accessToken, _ := env.service.GenerateAccessToken(user)
		_, err := env.service.ValidateRefreshToken(context.Background(), accessToken)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not refresh type")
		env.cache.AssertNotCalled(t, "Get")
	})

	t.Run("error: cache miss means token is expired or revoked", func(t *testing.T) {
		// Any cache error on the jti lookup must result in an expired/revoked error.
		env := newJWTTestEnv()
		jti := uuid.New().String()

		tokenStr := signedToken(jwt.MapClaims{
			"sub":   "user-1",
			"email": "user@example.com",
			"type":  "refresh",
			"jti":   jti,
			"exp":   time.Now().Add(time.Hour).Unix(),
		}, "test-secret")

		env.cache.On("Get", mock.Anything, "refresh:"+jti).Return("", errors.New("key not found")).Once()

		_, err := env.service.ValidateRefreshToken(context.Background(), tokenStr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired or revoked")
		env.cache.AssertExpectations(t)
	})

	t.Run("error: malformed token short-circuits before cache lookup", func(t *testing.T) {
		// Invalid JWT string must fail without any cache interaction.
		env := newJWTTestEnv()

		_, err := env.service.ValidateRefreshToken(context.Background(), "garbage")

		assert.Error(t, err)
		env.cache.AssertNotCalled(t, "Get")
	})
}

// ─── RevokeRefreshToken ──────────────────────────────────────────────────────

func TestRevokeRefreshToken(t *testing.T) {
	t.Run("success: valid refresh token is deleted from cache", func(t *testing.T) {
		// cache.Delete must be called with exactly "refresh:<jti>".
		env := newJWTTestEnv()
		jti := uuid.New().String()

		tokenStr := signedToken(jwt.MapClaims{
			"sub":   "user-1",
			"email": "user@example.com",
			"type":  "refresh",
			"jti":   jti,
			"exp":   time.Now().Add(time.Hour).Unix(),
		}, "test-secret")

		env.cache.On("Delete", mock.Anything, "refresh:"+jti).Return(nil).Once()

		err := env.service.RevokeRefreshToken(context.Background(), tokenStr)

		assert.NoError(t, err)
		env.cache.AssertExpectations(t)
	})

	t.Run("error: access token is rejected before cache delete", func(t *testing.T) {
		// Passing an access token must fail with "not refresh type" without touching cache.
		env := newJWTTestEnv()
		user := sampleUser()

		accessToken, _ := env.service.GenerateAccessToken(user)
		err := env.service.RevokeRefreshToken(context.Background(), accessToken)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not refresh type")
		env.cache.AssertNotCalled(t, "Delete")
	})

	t.Run("error: malformed token is rejected before cache delete", func(t *testing.T) {
		// A tampered or garbage token must fail without any cache interaction.
		env := newJWTTestEnv()

		err := env.service.RevokeRefreshToken(context.Background(), "garbage.token.here")

		assert.Error(t, err)
		env.cache.AssertNotCalled(t, "Delete")
	})
}
