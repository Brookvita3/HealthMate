package domain

import (
	"github.com/google/uuid"
)

type Permission struct {
	GroupId          uuid.UUID  `json:"group_id"`
	UserId           uuid.UUID  `json:"user_id"`
	MetricType       string     `json:"metric_type"`
	SharedWithUserId *uuid.UUID `json:"shared_with_user_id"`
}

// MemberVisibleMetrics holds the metric types that a viewer is allowed to see
// for a specific group member, after applying the two-tier sharing rule.
type MemberVisibleMetrics struct {
	MemberID uuid.UUID
	Metrics  []string
}
