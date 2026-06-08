package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func generateTestToken(secret []byte, tokenType string, exp time.Duration) string {
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "test@example.com",
		"role":  "user",
		"type":  tokenType,
		"exp":   time.Now().Add(exp).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)
	return tokenString
}

func TestJWTAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	r := gin.New()
	r.Use(JWTAuthMiddleware(secret))
	r.GET("/protected", func(c *gin.Context) {
		sub, _ := c.Get("sub")
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{
			"sub":  sub,
			"role": role,
		})
	})

	t.Run("Valid Token", func(t *testing.T) {
		token := generateTestToken([]byte(secret), "access", time.Hour)
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}
	})

	t.Run("Missing Header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Invalid Format", func(t *testing.T) {
		token := generateTestToken([]byte(secret), "access", time.Hour)
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", token) // Missing "Bearer "

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Invalid Token Type", func(t *testing.T) {
		token := generateTestToken([]byte(secret), "refresh", time.Hour)
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Wrong Secret", func(t *testing.T) {
		token := generateTestToken([]byte("wrong-secret"), "access", time.Hour)
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status Unauthorized, got %d", w.Code)
		}
	})
}
