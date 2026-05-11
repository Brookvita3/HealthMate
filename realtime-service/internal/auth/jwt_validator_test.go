package auth_test

import (
	"testing"
	"time"

	"realtime-service/internal/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const testSecret = "test-secret-key"

// makeToken creates a signed HS256 JWT with the given claims and secret.
func makeToken(secret string, claims jwt.MapClaims) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString([]byte(secret))
	return s
}

// validClaims returns a valid set of access-token claims.
func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub":   "user-uuid-1",
		"email": "test@example.com",
		"role":  "user",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
}

// ─── JWTValidator.ValidateToken ───────────────────────────────────────────────

func TestJWTValidator_ValidateToken(t *testing.T) {
	t.Run("success: returns correct UserID, Email, and Role from valid token", func(t *testing.T) {
		// A freshly signed token with all required claims must be accepted and
		// the extracted TokenClaims must match the original payload.
		v := auth.NewJWTValidator(testSecret)
		tokenStr := makeToken(testSecret, validClaims())

		claims, err := v.ValidateToken(tokenStr)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, "user-uuid-1", claims.UserID)
		assert.Equal(t, "test@example.com", claims.Email)
		assert.Equal(t, "user", claims.Role)
	})

	t.Run("error: expired token is rejected", func(t *testing.T) {
		// A token whose exp claim is in the past must be rejected by the validator.
		v := auth.NewJWTValidator(testSecret)
		tokenStr := makeToken(testSecret, jwt.MapClaims{
			"sub":   "user-1",
			"email": "test@example.com",
			"exp":   time.Now().Add(-time.Hour).Unix(),
		})

		claims, err := v.ValidateToken(tokenStr)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("error: token signed with wrong secret is rejected", func(t *testing.T) {
		// An HMAC signature produced with a different secret key must fail
		// verification even if the claims are otherwise valid.
		v := auth.NewJWTValidator(testSecret)
		tokenStr := makeToken("wrong-secret", validClaims())

		claims, err := v.ValidateToken(tokenStr)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("error: malformed token string is rejected without panicking", func(t *testing.T) {
		// Garbage input must return an error and must not panic.
		v := auth.NewJWTValidator(testSecret)

		claims, err := v.ValidateToken("not.a.valid.jwt")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("error: token with no sub claim returns 'missing user id'", func(t *testing.T) {
		// A token that omits the sub claim must be rejected with a descriptive error.
		v := auth.NewJWTValidator(testSecret)
		tokenStr := makeToken(testSecret, jwt.MapClaims{
			"email": "test@example.com",
			"role":  "user",
			"exp":   time.Now().Add(time.Hour).Unix(),
		})

		claims, err := v.ValidateToken(tokenStr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing user id")
		assert.Nil(t, claims)
	})

	t.Run("error: token with empty sub claim returns 'missing user id'", func(t *testing.T) {
		// An explicit empty string in sub must also be treated as missing.
		v := auth.NewJWTValidator(testSecret)
		tokenStr := makeToken(testSecret, jwt.MapClaims{
			"sub":   "",
			"email": "test@example.com",
			"exp":   time.Now().Add(time.Hour).Unix(),
		})

		claims, err := v.ValidateToken(tokenStr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing user id")
		assert.Nil(t, claims)
	})
}
