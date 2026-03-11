package metric

import (
	"context"
	"time"
)

type Service interface {
	RecordMetric(ctx context.Context, metric HealthMetric) error
	GetChartData(ctx context.Context, userID, metricType, timeRange string, start, end *time.Time) ([]MetricDataPoint, error)
}

type serviceImpl struct {
	repo MetricRepository
}

func NewMetricService(repo MetricRepository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) RecordMetric(ctx context.Context, metric HealthMetric) error {
	return s.repo.Store(ctx, &metric)
}

func (s *serviceImpl) GetChartData(ctx context.Context, userID, metricType, timeRange string, start, end *time.Time) ([]MetricDataPoint, error) {
	var startTime, endTime time.Time

	switch timeRange {
	case RangeLast7Days:
		endTime = time.Now()
		startTime = endTime.AddDate(0, 0, -7)
	case RangeLast30Days:
		endTime = time.Now()
		startTime = endTime.AddDate(0, 0, -30)
	case RangeCustom:
		if start == nil || end == nil {
			return nil, ErrMissingTimes
		}
		startTime = *start
		endTime = *end
	default:
		// Default to 7 days if not provided
		endTime = time.Now()
		startTime = endTime.AddDate(0, 0, -7)
	}

	duration := endTime.Sub(startTime)
	var bucketSize string

	days := duration.Hours() / 24
	if days <= 2 {
		bucketSize = "1 hour"
	} else if days <= 30 {
		bucketSize = "1 day"
	} else if days <= 365 {
		bucketSize = "1 week"
	} else {
		return nil, ErrInvalidRange
	}

	return s.repo.GetAggregatedMetrics(ctx, userID, metricType, bucketSize, startTime, endTime)
}
