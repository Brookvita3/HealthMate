package metric

import "time"

type HealthMetric struct {
	UserID    string    `json:"user_id"`
	Type      string    `json:"metric_type"` // e.g., "heart_rate", "steps_count", "calories"
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

type MetricDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type Watcher struct {
	UserID    string
	UserName  string // Name of the watcher
	GroupName string // Group through which the metric is shared
}

const (
	RangeLast24Hours = "24h"
	RangeLast7Days   = "7d"
	RangeLast30Days  = "30d"
	RangeCustom     = "custom"
)
