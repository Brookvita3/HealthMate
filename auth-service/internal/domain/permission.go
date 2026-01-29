package domain

import (
	"github.com/google/uuid"
)

type Permission struct {
	GroupId    uuid.UUID `json:"group_id"`
	UserId     uuid.UUID `json:"user_id"`
	MetricType string    `json:"metric_type"`
}
