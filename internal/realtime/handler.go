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

func (h *Handler) ServeWsGin(c *gin.Context) {
	userIdStr := c.Query("userId")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection for user %s: %v", userIdStr, err)
		return
	}

	client := NewClient(h.manager, conn, userIdStr)
	h.manager.register <- client

	go client.writePump()
	go client.readPump()
}
