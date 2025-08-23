package realtime

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// Client represents a WebSocket connection of the user.
type Client struct {
	manager *Manager
	conn    *websocket.Conn
	userID  string
	send    chan *OutgoingMessage
}

func NewClient(manager *Manager, conn *websocket.Conn, userID string) *Client {
	return &Client{
		manager: manager,
		conn:    conn,
		userID:  userID,
		send:    make(chan *OutgoingMessage, 256),
	}
}

// readPump listens for messages from the client.
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		var msg IncomingMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			log.Printf("error reading json: %v", err)
			break
		}
		c.manager.processMessage(c, &msg)
	}
}

// writePump sends messages from the server to the client.
func (c *Client) writePump() {
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
			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("error writing json: %v", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
