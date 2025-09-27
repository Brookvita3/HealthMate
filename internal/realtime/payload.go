package realtime

import "encoding/json"

// Action defines the type for incoming WebSocket message actions.
type Action string

// Constants for all supported client actions.
const (
	ActionSubscribeToUserData     Action = "subscribe_to_user_data"
	ActionUnsubscribeFromUserData Action = "unsubscribe_from_user_data"
	ActionWearableDataUpdate      Action = "wearable_data_update"
)

// Event defines the type for outgoing WebSocket message events.
type Event string

// Constants for all supported server events.
const (
	EventUserDataReceived  Event = "user_data_received"
	EventError             Event = "error"
	EventSubscribedSuccess Event = "subscribed_success"
)

// IncomingMessage is the structure for all messages sent from the client.
type IncomingMessage struct {
	Action  Action          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

// OutgoingMessage is the structure for all messages sent from the server.
type OutgoingMessage struct {
	Event   Event       `json:"event"`
	Payload interface{} `json:"payload"`
}

// UserTargetPayload is the payload for actions that target a specific user.
type UserTargetPayload struct {
	TargetUserId string `json:"target_user_id"`
}

// WearableDataPayload is the payload when the client sends wearable data.
type WearableDataPayload struct {
	Data json.RawMessage `json:"data"`
}

// UserDataBroadcastPayload is the payload for the user_data_received event.
type UserDataBroadcastPayload struct {
	FromUserId string          `json:"from_user_id"`
	Data       json.RawMessage `json:"data"`
}

// ErrorPayload is the payload for the error event.
type ErrorPayload struct {
	Message string `json:"message"`
	Action  Action `json:"action,omitempty"`
}

// Payload cho event subscribed_success
type SubscribedSuccessPayload struct {
	TargetUserId string `json:"target_user_id"`
}
