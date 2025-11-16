package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// ClientMessage is the struct for client message
type ClientMessage struct {
	Action string          `json:"action"  validate:"required"` // "subscribe", "unsubscribe"
	Items  []SubscribeItem `json:"items" validate:"required,min=1"`
}

// SubscribeItem is the struct for subscribe item
type SubscribeItem struct {
	TargetUserID string `json:"target_user_id"  validate:"required"`
	MetricType   string `json:"metric_type"  validate:"required"` // "heart_rate", "steps_count", "calories_burned"
}

// ServerMessage is the struct for server message
type ServerMessage struct {
	Type    string `json:"type"`    // "error", "success", "metric"
	Payload string `json:"payload"` // JSON string
}

// Client is client connection for websocket
type Client struct {
	id       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	viewerId string

	// Map: targetUserID -> Map: metricType -> bool
	// Save permission of client
	permissions map[string]map[string]bool // user-A-id -> {"heart_rate": true, "steps_count": true}
}

func NewClient(hub *Hub, conn *websocket.Conn, viewerID string) *Client {
	return &Client{
		id:          uuid.NewString(),
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		viewerId:    viewerID,
		permissions: make(map[string]map[string]bool),
	}
}

// readPump read message from client
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error readPump: %v", err)
			}
			break
		}
		c.handleClientMessage(ctx, message)
	}
}

// writePump write message to client
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleClientMessage handle message from client ( "subscribe", "unsubscribe" )
func (c *Client) handleClientMessage(ctx context.Context, message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshal message from client %s: %v", c.id, err)
		c.sendError("Unmarshal error")
		return
	}

	validate := validator.New()
	if err := validate.Struct(msg); err != nil {
		log.Printf("Error validate client message: %v", err)
		c.sendError("Validate client message error")
		return
	}

	switch msg.Action {
	case "subscribe":
		for _, item := range msg.Items {
			hasPerm, err := c.hub.permRepo.CheckAccess(
				ctx,
				c.viewerId,
				item.TargetUserID,
				item.MetricType,
			)

			if err != nil {
				c.sendError("Permission check error")
				continue
			}

			if !hasPerm {
				c.sendError(fmt.Sprintf("No permission for %s/%s", item.TargetUserID, item.MetricType))
				continue
			}

			// Save permission
			if _, ok := c.permissions[item.TargetUserID]; !ok {
				c.permissions[item.TargetUserID] = make(map[string]bool)
			}
			c.permissions[item.TargetUserID][item.MetricType] = true

			c.hub.subscribe <- SubscriptionEvent{
				Client:       c,
				TargetUserID: item.TargetUserID,
				MetricType:   item.MetricType,
			}
		}

		c.sendSuccess("Subscribe success")
		return

		// TODO: "unsubscribe"
	case "unsubscribe":
		for _, item := range msg.Items {
			c.hub.unsubscribe <- SubscriptionEvent{
				Client:       c,
				TargetUserID: item.TargetUserID,
				MetricType:   item.MetricType,
			}
		}
		c.sendSuccess("Unsubscribe success")
		return
	}
}

// CanView check if client can view metric by cache
func (c *Client) CanView(targetUserID string, metricType string) bool {
	if _, ok := c.permissions[targetUserID]; !ok {
		return false
	}
	if _, ok := c.permissions[targetUserID][metricType]; !ok {
		return false
	}
	return true
}

// sendError helper
func (c *Client) sendError(payload string) {
	msg, _ := json.Marshal(ServerMessage{Type: "error", Payload: payload})
	c.send <- msg
}

// sendSuccess helper
func (c *Client) sendSuccess(payload string) {
	msg, _ := json.Marshal(ServerMessage{Type: "success", Payload: payload})
	c.send <- msg
}
