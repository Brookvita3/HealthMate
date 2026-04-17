package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"realtime-service/internal/common"
	"realtime-service/internal/metric"
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

// ClientMessage represents a control message sent from the client
type ClientMessage struct {
	Action string          `json:"action"  validate:"required"` // "subscribe", "unsubscribe"
	Items  []SubscribeItem `json:"items" validate:"required,min=1"`
}

// SubscribeItem represents a single subscription request
type SubscribeItem struct {
	TargetUserID string `json:"target_user_id"  validate:"required"`
	MetricType   string `json:"metric_type"  validate:"required"` // "heart_rate", "steps_count", "calories_burned"
}

// ServerMessage represents a message sent from server to client
type ServerMessage struct {
	Type    string      `json:"type"`    // "error", "success", "metric"
	Payload interface{} `json:"payload"` // Payload can be string or object
}

// Client represents a WebSocket client connection
type Client struct {
	id       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	viewerId string

	// Map: targetUserID -> Map: metricType -> bool
	// Used to cache client permissions
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

// readPump reads messages from the WebSocket connection
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

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

// writePump writes messages to the WebSocket connection
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

// handleClientMessage routes messages from the client
// It distinguishes between control commands and metric data
func (c *Client) handleClientMessage(ctx context.Context, message []byte) {
	var raw map[string]interface{}
	if err := json.Unmarshal(message, &raw); err != nil {
		log.Printf("Error unmarshal raw message from client %s: %v", c.id, err)
		c.sendError("Invalid JSON format")
		return
	}

	// 1. Check if this is a control command (subscribe/unsubscribe)
	if _, ok := raw["action"]; ok {
		c.handleControlAction(ctx, message)
		return
	}

	// 2. Check if this is metric data pushed from a device/target user
	if _, ok := raw["user_id"]; ok {
		c.handleMetricPush(message)
		return
	}

	c.sendError("Unknown message structure")
}

// handleControlAction processes subscribe/unsubscribe commands
func (c *Client) handleControlAction(ctx context.Context, message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.sendError("Invalid control message format")
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
		hasError := false
		for _, item := range msg.Items {
			hasPerm, err := c.hub.permRepo.CheckAccess(
				ctx,
				c.viewerId,
				item.TargetUserID,
				item.MetricType,
			)

			if err != nil {
				log.Printf("Error checking permission for %s vs %s: %v", c.viewerId, item.TargetUserID, err)
				if bErr, ok := err.(*common.BusinessError); ok {
					c.sendError(bErr.Message)
				} else {
					c.sendError("Permission check error")
				}
				hasError = true
				continue
			}

			if !hasPerm {
				c.sendError(fmt.Sprintf("No permission for %s/%s", item.TargetUserID, item.MetricType))
				hasError = true
				continue
			}

			// Cache permission locally
			if _, ok := c.permissions[item.TargetUserID]; !ok {
				c.permissions[item.TargetUserID] = make(map[string]bool)
			}
			c.permissions[item.TargetUserID][item.MetricType] = true
			log.Printf("[Client %s] Cached subscription permission for %s/%s", c.id, item.TargetUserID, item.MetricType)

			c.hub.subscribe <- SubscriptionEvent{
				Client:       c,
				TargetUserID: item.TargetUserID,
				MetricType:   item.MetricType,
			}
		}

		if !hasError {
			c.sendSuccess("Subscribe success")
		}
		return

	case "unsubscribe":
		for _, item := range msg.Items {
			c.hub.unsubscribe <- SubscriptionEvent{
				Client:       c,
				TargetUserID: item.TargetUserID,
				MetricType:   item.MetricType,
			}
			// Remove from local cache
			if _, ok := c.permissions[item.TargetUserID]; ok {
				delete(c.permissions[item.TargetUserID], item.MetricType)
			}
		}
		c.sendSuccess("Unsubscribe success")
		return
	}
}

// handleMetricPush processes metric data pushed from the client
// and broadcasts it to subscribed observers
func (c *Client) handleMetricPush(message []byte) {
	var m metric.HealthMetric
	if err := json.Unmarshal(message, &m); err != nil {
		log.Printf("Error unmarshal metric push from %s: %v", c.viewerId, err)
		c.sendError("Invalid metric data format. Note: timestamp must be RFC3339 (e.g. 2026-01-02T15:04:05Z)")
		return
	}

	// Security check:
	// A client is only allowed to push data for its own UserID
	if m.UserID != c.viewerId {
		log.Printf("Security warning: Client %s tried to push data for user %s", c.viewerId, m.UserID)
		c.sendError("Forbidden: You can only push data for your own UserID")
		return
	}

	// If timestamp is not provided, use current time
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}

	// Publish metric to Kafka
	// The consumer will then pick it up and broadcast it via the Hub
	if err := c.hub.GetProducer().PublishMetric(context.Background(), &m); err != nil {
		log.Printf("Error publishing metric to Kafka: %v", err)
		c.sendError("Failed to process metric data")
		return
	}
}

// CanView checks whether the client has permission
// to view a specific metric of a target user
func (c *Client) CanView(targetUserID string, metricType string) bool {
	if _, ok := c.permissions[targetUserID]; !ok {
		log.Printf("[Client %s] No permissions found for target user %s", c.id, targetUserID)
		return false
	}
	if _, ok := c.permissions[targetUserID][metricType]; !ok {
		log.Printf("[Client %s] No permission specifically for %s of user %s", c.id, metricType, targetUserID)
		return false
	}
	return true
}

// sendError sends an error message to the client
func (c *Client) sendError(payload string) {
	msg, _ := json.Marshal(ServerMessage{Type: "error", Payload: payload})
	c.send <- msg
}

// sendSuccess sends a success message to the client
func (c *Client) sendSuccess(payload string) {
	msg, _ := json.Marshal(ServerMessage{Type: "success", Payload: payload})
	c.send <- msg
}
