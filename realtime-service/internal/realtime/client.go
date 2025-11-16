package realtime

import (
	"context"
	"encoding/json"
	"log"
	"time"

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
	Action       string `json:"action"` // "subscribe", "unsubscribe"
	TargetUserID string `json:"target_user_id"`
	MetricType   string `json:"metric_type"` // "heart_rate", "steps_count", "all"
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

// readPump read message from client
func (c *Client) readPump() {
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
		c.handleClientMessage(message)
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
func (c *Client) handleClientMessage(message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshal message from client %s: %v", c.id, err)
		return
	}

	switch msg.Action {
	case "subscribe":
		// 3. Check permision
		//hasPerm, err := c.hub.permRepo.CheckAccess(context.Background(), c.viewerID, msg.TargetUserID, msg.MetricType)
		hasPerm := true
		// if err != nil {
		// 	log.Printf("Error check permision from client %s: %v", c.id, err)
		// 	c.sendError("Error check permision from client")
		// 	return
		// }

		if !hasPerm {
			log.Printf("Client %s (User %s) no permission to view %s of User %s", c.id, c.viewerId, msg.MetricType, msg.TargetUserID)
			c.sendError("No permission to view metric")
			return
		}

		// 4. Save permission
		if _, ok := c.permissions[msg.TargetUserID]; !ok {
			log.Printf("Client %s (User %s) has permission to view %s of User %s", c.id, c.viewerId, msg.MetricType, msg.TargetUserID)
			c.permissions[msg.TargetUserID] = make(map[string]bool)
		}
		c.permissions[msg.TargetUserID][msg.MetricType] = true

		c.hub.subscribe <- SubscriptionEvent{
			Client:       c,
			TargetUserID: msg.TargetUserID,
			MetricType:   msg.MetricType,
		}

		c.sendSuccess("Subscribe success")
		return

		// TODO: "unsubscribe"
	}
}

// CanView check if client can view metric by cache
func (c *Client) CanView(targetUserID string, metricType string) bool {
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
