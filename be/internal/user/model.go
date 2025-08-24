package user

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

type User struct {
	ID      uuid.UUID   `json:"id" db:"id"`
	Email   string      `json:"email" db:"email"`
	Name    string      `json:"name" db:"name"`
	Picture pgtype.Text `json:"picture,omitempty" db:"picture"` // Allows NULL
	Role    string      `json:"role" db:"role"`
	Status  string      `json:"status" db:"status"`

	// Authentication fields, not returned via JSON
	Provider string      `json:"-" db:"provider"`
	GoogleID pgtype.Text `json:"-" db:"google_id"` // Allows NULL
	Password pgtype.Text `json:"-" db:"password"`  // Allows NULL

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
