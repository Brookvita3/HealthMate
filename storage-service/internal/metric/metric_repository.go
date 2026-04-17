package metric

import (
	"context"
	"time"
)

type MetricRepository interface {
	Store(ctx context.Context, metric *HealthMetric) error
	GetAggregatedMetrics(ctx context.Context, userID string, metricType string, bucketSize string, startTime, endTime time.Time) ([]MetricDataPoint, error)
	GetUserThresholds(ctx context.Context, userID string) ([]UserThreshold, error)
	GetThresholdByMetricName(ctx context.Context, userID string, metricName string) (*UserThreshold, error)
	UpsertThreshold(ctx context.Context, threshold *UserThreshold) error
	GetMetricWatchers(ctx context.Context, userID, metricType string) ([]Watcher, error)
	GetUserName(ctx context.Context, userID string) (string, error)
}
