package realtime

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}

// ServeWs forwards WebSocket request
func (h *Handler) ServeWs(c *gin.Context) {
	userId, ok := c.Get("userId")
	if !ok || userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDStr, _ := userId.(string)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection for user %s: %v", userIDStr, err)
		return
	}

	client := NewClient(h.manager, conn, userIDStr)
	h.manager.register <- client

	go client.writePump()
	go client.readPump()
}
