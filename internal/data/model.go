package data

import "github.com/google/uuid"

type UserHealth struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Picture  string    `json:"picture"`
	Provider string    `json:"provider"`
	Password string    `json:"password"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	GoogleID string    `json:"google_id"`
}
