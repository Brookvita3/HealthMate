package auth

import "github.com/google/uuid"

type User struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	GoogleID string    `json:"google_id"`
	Password string    `json:"password"`
	Picture  string    `json:"picture"`
	Provider string    `json:"provider"`
}
