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

	subscriptions map[string]map[string]map[string]map[*Client]bool // "targetUserID" -> "metricType" -> "groupID" -> [client_B, client_C]

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
	GroupID      string
}

func NewHub(permRepo permission.Repository, producer *kafka.Producer) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		subscriptions: make(map[string]map[string]map[string]map[*Client]bool),

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
			userSubs, ok := h.subscriptions[ev.TargetUserID]
			if !ok {
				userSubs = make(map[string]map[string]map[*Client]bool)
				h.subscriptions[ev.TargetUserID] = userSubs
			}
			metricSubs, ok := userSubs[ev.MetricType]
			if !ok {
				metricSubs = make(map[string]map[*Client]bool)
				userSubs[ev.MetricType] = metricSubs
			}
			groupSubs, ok := metricSubs[ev.GroupID]
			if !ok {
				groupSubs = make(map[*Client]bool)
				metricSubs[ev.GroupID] = groupSubs
			}
			groupSubs[ev.Client] = true

		case ev := <-h.unsubscribe:
			if userSubs, ok := h.subscriptions[ev.TargetUserID]; ok {
				if metricSubs, ok := userSubs[ev.MetricType]; ok {
					if groupSubs, ok := metricSubs[ev.GroupID]; ok {
						delete(groupSubs, ev.Client)
						if len(groupSubs) == 0 {
							delete(metricSubs, ev.GroupID)
						}
					}
					if len(metricSubs) == 0 {
						delete(userSubs, ev.MetricType)
					}
				}
				if len(userSubs) == 0 {
					delete(h.subscriptions, ev.TargetUserID)
				}
			}

		case m := <-h.broadcast:
			// log.Printf("[Hub] Broadcast received for UserID: %s, Metric: %s", m.UserID, m.Type)
			
			// 1. Fetch live authorized watchers from DB (Strict Base + Filter logic)
			// authorizedUserGroups maps watcherUserID -> []groupIDs that authorized them
			authorizedUserGroups, err := h.permRepo.GetMetricWatchers(context.Background(), m.UserID, m.Type)
			if err != nil {
				log.Printf("[Hub] Error fetching watchers for %s/%s: %v", m.UserID, m.Type, err)
				continue
			}

			if len(authorizedUserGroups) == 0 {
				continue
			}

			// 2. Fetch active subscribers for this user and metric type
			userSubs := h.subscriptions[m.UserID]
			if userSubs == nil {
				continue
			}
			metricSubs := userSubs[m.Type]
			if len(metricSubs) == 0 {
				continue
			}

			// 3. Prepare message
			msgBody := ServerMessage{
				Type:    "metric",
				Payload: m,
			}
			data, _ := json.Marshal(msgBody)

			// 4. Send only to clients who are BOTH subscribed and authorized in the same group
			sentCount := 0
			for groupID, clients := range metricSubs {
				for client := range clients {
					authorizedGroups := authorizedUserGroups[client.viewerId]
					isAuthorized := false

					if groupID == "" {
						// Global subscription (no specific group requested) -> OK if authorized by ANY group
						isAuthorized = len(authorizedGroups) > 0
					} else {
						// Group-isolated subscription -> OK ONLY if authorized by THIS specific group
						for _, g := range authorizedGroups {
							if g == groupID {
								isAuthorized = true
								break
							}
						}
					}

					if isAuthorized {
						select {
						case client.send <- data:
							sentCount++
						default:
							log.Printf("[Hub] Client %s buffer full, dropping message", client.id)
						}
					}
				}
			}
			if sentCount > 0 {
				log.Printf("[Hub] Broadcasted %s/%s to %d authorized subscribers", m.UserID, m.Type, sentCount)
			}
		}
	}
}

// removeClientSubscriptions deletes all subscriptions of a client
func (h *Hub) removeClientSubscriptions(client *Client) {
	for targetUserID, userSubs := range h.subscriptions {
		for metricType, metricSubs := range userSubs {
			for groupID, groupSubs := range metricSubs {
				if _, ok := groupSubs[client]; ok {
					delete(groupSubs, client)
					if len(groupSubs) == 0 {
						delete(metricSubs, groupID)
					}
				}
			}
			if len(metricSubs) == 0 {
				delete(userSubs, metricType)
			}
		}
		if len(userSubs) == 0 {
			delete(h.subscriptions, targetUserID)
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
