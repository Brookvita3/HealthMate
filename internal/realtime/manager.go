package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"healthmate/internal/common"
	"healthmate/internal/connection"

	"github.com/google/uuid"
)

// Manager handles all WebSocket clients and their subscriptions.
type Manager struct {
	clients       map[string]*Client
	subscriptions map[string]map[*Client]bool // Set structure
	mu            sync.RWMutex
	register      chan *Client
	unregister    chan *Client
	connSvc       connection.Service
}

// NewManager creates a new Manager instance.
func NewManager(connSvc connection.Service) *Manager {
	return &Manager{
		clients:       make(map[string]*Client),
		subscriptions: make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		connSvc:       connSvc,
	}
}

// Run starts the main event loop for the manager.
func (m *Manager) Run() {
	for {
		select {
		case client := <-m.register:
			m.handleRegister(client)
		case client := <-m.unregister:
			m.handleUnregister(client)
		}
	}
}

func (m *Manager) handleRegister(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[client.userId] = client
	log.Printf("Client registered: %s", client.userId)
}

func (m *Manager) handleUnregister(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[client.userId]; ok {
		delete(m.clients, client.userId)
		log.Printf("Client unregistered: %s", client.userId)
	}

	for targetuserId, listeners := range m.subscriptions {
		if _, ok := listeners[client]; ok {
			delete(listeners, client)
			if len(listeners) == 0 {
				delete(m.subscriptions, targetuserId)
			}
		}
	}

	close(client.send)
}

// processMessage routes incoming messages from clients to the appropriate handler.
func (m *Manager) processMessage(client *Client, msg *IncomingMessage) {
	switch msg.Action {
	case ActionSubscribeToUserData:
		m.handleSubscribe(client, msg)
	case ActionUnsubscribeFromUserData:
		m.handleUnsubscribe(client, msg)
	case ActionWearableDataUpdate:
		m.handleBroadcast(client, msg)
	default:
		log.Printf("Unknown action from user %s: %s", client.userId, msg.Action)
	}
}

// sendErrorToClient is a helper to send a structured error message back to a client.
func (m *Manager) sendErrorToClient(client *Client, err error, originalAction Action) {
	errorMsg := &OutgoingMessage{
		Event: EventError,
		Payload: ErrorPayload{
			Message: err.Error(),
			Action:  originalAction,
		},
	}
	// Use a non-blocking send to avoid deadlocks if the client's send channel is full.
	select {
	case client.send <- errorMsg:
	default:
		log.Printf("Failed to send error to client %s: send channel is full.", client.userId)
	}
}

// handleSubscribe processes a subscription request from a client.
func (m *Manager) handleSubscribe(client *Client, msg *IncomingMessage) {
	var p UserTargetPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		log.Printf("Error unmarshalling subscribe payload: %v", err)
		m.sendErrorToClient(client, ErrInvalidPayloadFormat, msg.Action)
		return
	}

	viewerId, err := uuid.Parse(client.userId)
	if err != nil {
		log.Printf("Internal Error: Invalid viewer userId in client struct: %v", err)
		m.sendErrorToClient(client, common.ErrInternalServer, msg.Action) // Keep this generic for security
		return
	}

	targetId, err := uuid.Parse(p.TargetUserId)
	if err != nil {
		log.Printf("Internal Error: Invalid target userId in payload: %v", err)
		m.sendErrorToClient(client, common.ErrInvalidUUIDFormat, msg.Action)
		return
	}

	// Permission check using the ConnectionService
	conn, err := m.connSvc.GetConnectionByPair(context.Background(), viewerId, targetId)
	if err != nil {
		if errors.Is(err, connection.ErrConnectionNotFound) {
			m.sendErrorToClient(client, ErrPermissionDenied, msg.Action)
			return
		}
		log.Printf("Error checking connection status for %s -> %s: %v", viewerId, targetId, err)
		m.sendErrorToClient(client, common.ErrInternalServer, msg.Action)
		return
	}

	if conn.Status != connection.StatusAccepted {
		m.sendErrorToClient(client, ErrPermissionDenied, msg.Action)
		return
	}

	log.Printf("Client %s subscribing to %s", client.userId, p.TargetUserId)
	m.mu.Lock()
	if _, ok := m.subscriptions[p.TargetUserId]; !ok {
		m.subscriptions[p.TargetUserId] = make(map[*Client]bool)
	}
	m.subscriptions[p.TargetUserId][client] = true
	m.mu.Unlock()

	// 2. Create and send success message
	successMsg := &OutgoingMessage{
		Event:   EventSubscribedSuccess,
		Payload: SubscribedSuccessPayload(p),
	}

	select {
	case client.send <- successMsg:
	default:
		log.Printf("Failed to send subscription success message to client %s: send channel is full.", client.userId)
	}
}

// handleUnsubscribe processes an unsubscription request from a client.
func (m *Manager) handleUnsubscribe(client *Client, msg *IncomingMessage) {
	var p UserTargetPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		log.Printf("Error unmarshalling unsubscribe payload: %v", err)
		return
	}

	log.Printf("Client %s unsubscribing from %s", client.userId, p.TargetUserId)

	m.mu.Lock()
	defer m.mu.Unlock()

	if listeners, ok := m.subscriptions[p.TargetUserId]; ok {
		delete(listeners, client)
		if len(listeners) == 0 {
			delete(m.subscriptions, p.TargetUserId)
		}
	}
}

// handleBroadcast forwards a message from a sender to all subscribed listeners.
func (m *Manager) handleBroadcast(sender *Client, msg *IncomingMessage) {
	var p WearableDataPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		log.Printf("Error unmarshalling data payload: %v", err)
		return
	}

	m.mu.RLock()
	listeners, ok := m.subscriptions[sender.userId]
	m.mu.RUnlock()

	if !ok {
		return
	}

	broadcastMsg := &OutgoingMessage{
		Event: EventUserDataReceived,
		Payload: UserDataBroadcastPayload{
			FromUserId: sender.userId,
			Data:       p.Data,
		},
	}

	for listener := range listeners {
		select {
		case listener.send <- broadcastMsg:
		default:

		}
	}
}
