package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`

	// Allows NULL
	Picture string `json:"picture,omitempty"`
	Role    string `json:"role"`   // "user"
	Status  string `json:"status"` // "unverified", "verified"

	Provider string `json:"-"`
	// Allows NULL
	GoogleID string `json:"-"`

	Password string `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
