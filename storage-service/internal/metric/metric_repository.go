package metric

import (
	"context"
)

type MetricRepository interface {
	Store(ctx context.Context, metric *HealthMetric) error
}
