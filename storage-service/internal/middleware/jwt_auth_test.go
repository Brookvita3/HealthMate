package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"storage-service/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// makeAccessToken returns a signed HS256 JWT with the given claims.
func makeAccessToken(secret string, claims jwt.MapClaims) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString([]byte(secret))
	return s
}

// validClaims returns a baseline set of access-token claims.
func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub":   "user-uuid-1",
		"email": "test@example.com",
		"role":  "user",
		"type":  "access",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
}

// serve runs a single GET /test request through the middleware and returns the recorder.
// authHeader is set verbatim on the Authorization header; pass "" to omit it entirely.
func serve(secret, authHeader string) *httptest.ResponseRecorder {
	r := gin.New()
	r.Use(middleware.JWTAuthMiddleware(secret))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	r.ServeHTTP(w, req)
	return w
}

// serveCapture is like serve but also returns the gin context values set by the middleware.
func serveCapture(secret, authHeader string) (*httptest.ResponseRecorder, map[string]string) {
	r := gin.New()
	r.Use(middleware.JWTAuthMiddleware(secret))
	captured := map[string]string{}
	r.GET("/test", func(c *gin.Context) {
		for _, k := range []string{"sub", "email", "role"} {
			if v, ok := c.Get(k); ok {
				captured[k] = v.(string)
			}
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	r.ServeHTTP(w, req)
	return w, captured
}

// ─── JWTAuthMiddleware ────────────────────────────────────────────────────────

func TestJWTAuthMiddleware(t *testing.T) {
	const secret = "test-secret"

	t.Run("success: valid access token passes and sets sub/email/role in context", func(t *testing.T) {
		// A properly signed access token must let the request through (200) and
		// make sub, email, and role available in the Gin context.
		tokenStr := makeAccessToken(secret, validClaims())

		w, ctx := serveCapture(secret, "Bearer "+tokenStr)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "user-uuid-1", ctx["sub"])
		assert.Equal(t, "test@example.com", ctx["email"])
		assert.Equal(t, "user", ctx["role"])
	})

	t.Run("error: missing Authorization header returns 401", func(t *testing.T) {
		// Requests without an Authorization header must be rejected.
		w := serve(secret, "")

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error: header without Bearer scheme returns 401", func(t *testing.T) {
		// A token sent without the 'Bearer' prefix (e.g., bare token or wrong scheme)
		// must be rejected because the format is invalid.
		tokenStr := makeAccessToken(secret, validClaims())

		w := serve(secret, tokenStr) // no "Bearer " prefix

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error: token signed with wrong secret returns 401", func(t *testing.T) {
		// A token whose HMAC was produced with a different key must be rejected.
		tokenStr := makeAccessToken("wrong-secret", validClaims())

		w := serve(secret, "Bearer "+tokenStr)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error: expired token returns 401", func(t *testing.T) {
		// A token whose exp is in the past must be rejected by the middleware.
		claims := validClaims()
		claims["exp"] = time.Now().Add(-time.Hour).Unix()
		tokenStr := makeAccessToken(secret, claims)

		w := serve(secret, "Bearer "+tokenStr)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error: refresh token (wrong type) returns 401", func(t *testing.T) {
		// Tokens with type != "access" must be rejected even if the signature is valid.
		// This prevents refresh tokens from being used as bearer auth.
		claims := validClaims()
		claims["type"] = "refresh"
		tokenStr := makeAccessToken(secret, claims)

		w := serve(secret, "Bearer "+tokenStr)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error: malformed token string returns 401", func(t *testing.T) {
		// Garbage input in the Authorization header must be rejected without panicking.
		w := serve(secret, "Bearer not.a.valid.jwt")

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
