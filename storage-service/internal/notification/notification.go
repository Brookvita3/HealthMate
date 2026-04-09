package notification

import (
	"context"
	"time"
)

type Notification struct {
	Title string
	Body  string
	Data  map[string]string
}

type Service interface {
	SendToUser(ctx context.Context, userID string, notification Notification) error
	SendToToken(ctx context.Context, token string, notification Notification) error
}

type DeviceToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository interface {
	SaveToken(ctx context.Context, token DeviceToken) error
	GetTokensByUserID(ctx context.Context, userID string) ([]string, error)
	DeleteToken(ctx context.Context, userID string, token string) error
}
