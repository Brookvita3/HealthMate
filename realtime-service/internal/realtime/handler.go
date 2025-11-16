package realtime

import (
	"context"
	"log"
	"net/http"
	"realtime-service/internal/permission"
	authpb "realtime-service/proto/pb"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Handler struct {
	authClient authpb.AuthServiceClient
	permRepo   permission.Repository
	hub        *Hub
}

func NewHandler(authClient authpb.AuthServiceClient, permRepo permission.Repository, hub *Hub) *Handler {
	return &Handler{authClient: authClient, permRepo: permRepo, hub: hub}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing Authorization header", http.StatusUnauthorized)
		return
	}

	resp, err := h.authClient.ValidateToken(context.Background(), &authpb.ValidateTokenRequest{Token: token})
	if err != nil || !resp.Valid {
		http.Error(w, "invalid Authorization header", http.StatusUnauthorized)
		return
	}

	h.upgradeToWebSocket(w, r, resp.UserId)
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

	client := &Client{
		id:          uuid.NewString(),
		hub:         h.hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		viewerId:    viewerID,
		permissions: make(map[string]map[string]bool),
	}

	h.hub.register <- client

	ctx := context.Background()
	go client.writePump(ctx)
	go client.readPump()
}
