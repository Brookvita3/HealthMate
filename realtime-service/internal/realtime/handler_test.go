package realtime_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"realtime-service/internal/auth"
	"realtime-service/internal/realtime"

	"github.com/stretchr/testify/assert"
)

// mockTokenValidator is a simple function-based mock for auth.TokenValidator.
type mockTokenValidator struct {
	validateFunc func(token string) (*auth.TokenClaims, error)
}

func (m *mockTokenValidator) ValidateToken(token string) (*auth.TokenClaims, error) {
	return m.validateFunc(token)
}

// ─── Handler.ServeHTTP ────────────────────────────────────────────────────────

func TestHandler_ServeHTTP_MissingToken(t *testing.T) {
	t.Run("returns 401 when token query param is absent", func(t *testing.T) {
		// With no ?token= in the URL the handler must reject the request immediately
		// without ever calling ValidateToken.
		v := &mockTokenValidator{
			validateFunc: func(token string) (*auth.TokenClaims, error) {
				t.Fatal("ValidateToken must not be called when token is missing")
				return nil, nil
			},
		}
		h := realtime.NewHandler(v, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandler_ServeHTTP_InvalidToken(t *testing.T) {
	t.Run("returns 401 when ValidateToken returns an error", func(t *testing.T) {
		// Any token string that the validator rejects must produce a 401 response.
		v := &mockTokenValidator{
			validateFunc: func(token string) (*auth.TokenClaims, error) {
				return nil, errors.New("invalid token")
			},
		}
		h := realtime.NewHandler(v, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/ws?token=bad-token", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandler_ServeHTTP_ValidToken_AttemptsUpgrade(t *testing.T) {
	t.Run("proceeds past auth (non-401) when token is valid", func(t *testing.T) {
		// A valid token must pass the auth check and reach the WebSocket upgrade.
		// In an httptest context the gorilla upgrader fails the handshake (400),
		// but the response must NOT be 401 — proving auth was accepted.
		// The Hub is nil here because upgrader.Upgrade returns an error before any
		// hub interaction occurs.
		v := &mockTokenValidator{
			validateFunc: func(token string) (*auth.TokenClaims, error) {
				return &auth.TokenClaims{UserID: "user-1", Email: "u@example.com", Role: "user"}, nil
			},
		}
		h := realtime.NewHandler(v, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/ws?token=valid-token", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	})
}
