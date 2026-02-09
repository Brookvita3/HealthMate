package domain

import (
	"time"

	"github.com/google/uuid"
)

type MetricType struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
