package heathdata

import "github.com/google/uuid"

type UserHealth struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}
