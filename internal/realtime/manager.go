package realtime

import (
	"encoding/json"
	"log"
	"sync"
)

// Manager manages all clients and subscriptions.
type Manager struct {
	clients       map[string]*Client
	subscriptions map[string]map[*Client]bool // Set structure
	mu            sync.RWMutex

	register   chan *Client
	unregister chan *Client
}

func NewManager() *Manager {
	return &Manager{
		clients:       make(map[string]*Client),
		subscriptions: make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
	}
}

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
	m.clients[client.userID] = client
	log.Printf("Client registered: %s", client.userID)
}

func (m *Manager) handleUnregister(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove client from the management list
	if _, ok := m.clients[client.userID]; ok {
		delete(m.clients, client.userID)
		log.Printf("Client unregistered: %s", client.userID)
	}

	// Remove this client from all subscriptions it is currently subscribed to.
	for targetUserID, listeners := range m.subscriptions {
		if _, ok := listeners[client]; ok {
			delete(listeners, client)
			if len(listeners) == 0 {
				delete(m.subscriptions, targetUserID)
			}
		}
	}

	close(client.send)
}

func (m *Manager) processMessage(client *Client, msg *IncomingMessage) {
	switch msg.Action {
	case "subscribe_to_user_data":
		m.handleSubscribe(client, msg.Payload)
	case "unsubscribe_from_user_data":
		m.handleUnsubscribe(client, msg.Payload)
	case "wearable_data_update":
		m.handleBroadcast(client, msg.Payload)
	default:
		log.Printf("Unknown action: %s", msg.Action)
	}
}

func (m *Manager) handleSubscribe(client *Client, payload json.RawMessage) {
	var p SubscribePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("Error unmarshalling subscribe payload: %v", err)
		return
	}

	// Check member belongs to group (Skipped in this example for simplicity)
	log.Printf("Client %s subscribing to %s", client.userID, p.TargetUserID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.subscriptions[p.TargetUserID]; !ok {
		m.subscriptions[p.TargetUserID] = make(map[*Client]bool)
	}
	m.subscriptions[p.TargetUserID][client] = true
}

func (m *Manager) handleUnsubscribe(client *Client, payload json.RawMessage) {
	var p SubscribePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("Error unmarshalling unsubscribe payload: %v", err)
		return
	}

	log.Printf("Client %s unsubscribing from %s", client.userID, p.TargetUserID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if listeners, ok := m.subscriptions[p.TargetUserID]; ok {
		delete(listeners, client)
		if len(listeners) == 0 {
			delete(m.subscriptions, p.TargetUserID)
		}
	}
}

func (m *Manager) handleBroadcast(sender *Client, payload json.RawMessage) {
	var p WearableDataPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		log.Printf("Error unmarshalling data payload: %v", err)
		return
	}

	m.mu.RLock()
	listeners, ok := m.subscriptions[sender.userID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	broadcastMsg := &OutgoingMessage{
		Event: "user_data_received",
		Payload: UserDataBroadcastPayload{
			FromUserID: sender.userID,
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
