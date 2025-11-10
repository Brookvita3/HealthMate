package metric

import "context"

type Service interface {
	RecordMetric(ctx context.Context, metric HealthMetric) error
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
