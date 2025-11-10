package kafka

import "time"

type KafkaMessage struct {
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"` // e.g., "heart_rate", "steps_count", "calories"
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}
