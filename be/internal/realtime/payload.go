package realtime

import "encoding/json"

// Structure for all messages sent by the client
type IncomingMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

// Structure for all messages sent from the server to the client
type OutgoingMessage struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// Payload when the client wants to subscribe/unsubscribe
type SubscribePayload struct {
	TargetUserID string `json:"target_user_id"`
	GroupID      string `json:"group_id"` // check group
}

// Payload when the client sends wearable data
type WearableDataPayload struct {
	Data json.RawMessage `json:"data"` // raw data
}

// Payload sent by the server when new data is available
type UserDataBroadcastPayload struct {
	FromUserID string          `json:"from_user_id"`
	Data       json.RawMessage `json:"data"`
}
