package metric

import "time"

type UserThreshold struct {
	UserID     string    `json:"user_id"`
	MetricID   string    `json:"metric_id"`
	MetricName string    `json:"metric_name"` // For UI display convenience
	MinValue   *float64  `json:"min_value,omitempty"`
	MaxValue   *float64  `json:"max_value,omitempty"`
	IsEnabled  bool      `json:"is_enabled"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ptr(f float64) *float64 { return &f }
