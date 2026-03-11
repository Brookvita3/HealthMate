package metric

import (
	"context"
	"time"
)

type MetricRepository interface {
	Store(ctx context.Context, metric *HealthMetric) error
	GetAggregatedMetrics(ctx context.Context, userID string, metricType string, bucketSize string, startTime, endTime time.Time) ([]MetricDataPoint, error)
}
