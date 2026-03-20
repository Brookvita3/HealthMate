package domain

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

	Phone      string  `json:"phone,omitempty"`
	Address    string  `json:"address,omitempty"`
	Gender     string  `json:"gender,omitempty"`
	Birthday   string  `json:"birthday,omitempty"`
	Weight     float64 `json:"weight,omitempty"`
	Height     float64 `json:"height,omitempty"`
	BloodGroup string  `json:"blood_group,omitempty"`

	Provider string `json:"-"`
	// Allows NULL
	GoogleID string `json:"-"`

	Password string `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
