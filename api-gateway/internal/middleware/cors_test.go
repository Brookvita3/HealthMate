package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	// Define an OPTIONS handler to ensure middleware intercepts it
	r.OPTIONS("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach here")
	})

	t.Run("GET request allows origin", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Errorf("Expected origin http://example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("OPTIONS request is intercepted", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Errorf("Expected origin http://example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}
