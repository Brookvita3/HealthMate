package realtime

import (
	"context"
	"log"
	"net/http"
	"realtime-service/internal/auth"
	"realtime-service/internal/permission"

	"github.com/gorilla/websocket"
)

type Handler struct {
	tokenValidator auth.TokenValidator
	permRepo       permission.Repository
	hub            *Hub
}

func NewHandler(tokenValidator auth.TokenValidator, permRepo permission.Repository, hub *Hub) *Handler {
	return &Handler{tokenValidator: tokenValidator, permRepo: permRepo, hub: hub}
}

// ServeHTTP handles the WebSocket upgrade request.
// @Summary Webhook connection
// @Description Upgrade to WebSocket connection for real-time metrics
// @Tags realtime
// @Param token query string true "Authentication token"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {string} string "Unauthorized"
// @Router /ws [get]
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing Authorization header", http.StatusUnauthorized)
		return
	}

	claims, err := h.tokenValidator.ValidateToken(token)
	if err != nil {
		http.Error(w, "invalid Authorization header", http.StatusUnauthorized)
		return
	}

	h.upgradeToWebSocket(w, r, claims.UserID)
}

func (h *Handler) upgradeToWebSocket(w http.ResponseWriter, r *http.Request, viewerID string) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "" || origin == "https://healthmate.com"

		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := NewClient(h.hub, conn, viewerID)

	h.hub.register <- client

	ctx := context.Background()
	go client.writePump(ctx)
	go client.readPump(ctx)
}
