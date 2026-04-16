package realtime

import (
	"context"
	"encoding/json"
	"log"
	"realtime-service/internal/kafka"
	"realtime-service/internal/metric"
	"realtime-service/internal/permission"
)

type Hub struct {
	clients map[*Client]bool

	subscriptions map[string]map[*Client]bool // "user-A-id" -> [client_B, client_C]

	register    chan *Client
	unregister  chan *Client
	subscribe   chan SubscriptionEvent
	unsubscribe chan SubscriptionEvent
	broadcast   chan *metric.HealthMetric

	permRepo permission.Repository
	producer *kafka.Producer
}

type SubscriptionEvent struct {
	Client       *Client
	TargetUserID string
	MetricType   string
}

func NewHub(permRepo permission.Repository, producer *kafka.Producer) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		subscriptions: make(map[string]map[*Client]bool),

		register:   make(chan *Client),
		unregister: make(chan *Client),

		subscribe:   make(chan SubscriptionEvent),
		unsubscribe: make(chan SubscriptionEvent),

		broadcast: make(chan *metric.HealthMetric),

		permRepo: permRepo,
		producer: producer,
	}
}

// Run runs the hub
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				h.removeClientSubscriptions(client)
				close(client.send)
			}

		case ev := <-h.subscribe:
			subs, ok := h.subscriptions[ev.TargetUserID]
			if !ok {
				subs = make(map[*Client]bool)
				h.subscriptions[ev.TargetUserID] = subs
			}
			subs[ev.Client] = true

		case ev := <-h.unsubscribe:
			if subs, ok := h.subscriptions[ev.TargetUserID]; ok {
				delete(subs, ev.Client)
				if len(subs) == 0 {
					delete(h.subscriptions, ev.TargetUserID)
				}
			}

		case m := <-h.broadcast:
			log.Printf("[Hub] Broadcast received for UserID: %s, Metric: %s", m.UserID, m.Type)
			subs := h.subscriptions[m.UserID]
			if subs == nil {
				log.Printf("[Hub] No subscribers found for UserID: %s", m.UserID)
				continue
			}

			log.Printf("[Hub] Found %d potential subscribers for UserID: %s", len(subs), m.UserID)

			// Wrap metric in ServerMessage structure
			msgBody := ServerMessage{
				Type:    "metric",
				Payload: m,
			}
			data, _ := json.Marshal(msgBody)

			sentCount := 0
			for client := range subs {
				if client.CanView(m.UserID, m.Type) {
					select {
					case client.send <- data:
						sentCount++
					default:
						log.Printf("[Hub] Client %s send buffer full, dropping message", client.id)
					}
				} else {
					log.Printf("[Hub] Client %s permission denied for %s/%s", client.id, m.UserID, m.Type)
				}
			}
			log.Printf("[Hub] Successfully broadcasted to %d clients", sentCount)
		}
	}
}

// removeClientSubscriptions deletes all subscriptions of a client
func (h *Hub) removeClientSubscriptions(client *Client) {
	for targetUserID, subs := range h.subscriptions {
		if _, ok := subs[client]; ok {
			delete(subs, client)
			if len(subs) == 0 {
				delete(h.subscriptions, targetUserID)
			}
		}
	}
}

// BroadcastMetric sends a metric to all clients who are subscribed
func (h *Hub) BroadcastMetric(m *metric.HealthMetric) {
	h.broadcast <- m
}

// GetProducer returns the kafka producer
func (h *Hub) GetProducer() *kafka.Producer {
	return h.producer
}
