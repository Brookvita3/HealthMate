package metric

import (
	"context"
	"errors"
	"fmt"
	"log"
	"storage-service/internal/notification"
	"time"
)

type Service interface {
	RecordMetric(ctx context.Context, metric HealthMetric) error
	GetChartData(ctx context.Context, userID, metricType, timeRange string, start, end *time.Time) ([]MetricDataPoint, error)
	GetThresholds(ctx context.Context, userID string) ([]UserThreshold, error)
	SetUserThreshold(ctx context.Context, threshold UserThreshold) error
}

type serviceImpl struct {
	repo             MetricRepository
	notificationSvc notification.Service
}

func NewMetricService(repo MetricRepository, notificationSvc notification.Service) Service {
	return &serviceImpl{
		repo:             repo,
		notificationSvc: notificationSvc,
	}
}

func (s *serviceImpl) RecordMetric(ctx context.Context, metric HealthMetric) error {
	err := s.repo.Store(ctx, &metric)
	if err != nil {
		return err
	}

	// Check thresholds for spo2 and blood_pressure
	if metric.Type == "spo2" || metric.Type == "blood_pressure" {
		go s.checkThresholds(context.Background(), metric)
	}

	return nil
}

func (s *serviceImpl) checkThresholds(ctx context.Context, m HealthMetric) {
	// Create a timeout context for the threshold check and notification sending
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	t, err := s.repo.GetThresholdByMetricName(ctx, m.UserID, m.Type)
	if err != nil {
		if errors.Is(err, ErrMetricNotFound) || errors.Is(err, ErrUnknownMetricType) {
			// Expected if no threshold is configured for this metric
			return
		}
		log.Printf("Error retrieving threshold for user %s, metric %s: %v", m.UserID, m.Type, err)
		return
	}

	if t.IsEnabled {
		alert := false
		msg := ""

		if t.MinValue != nil && m.Value < *t.MinValue {
			alert = true
			msg = fmt.Sprintf("Cảnh báo: %s của bạn đang ở mức thấp (%.2f), ngưỡng tối thiểu là %.2f", m.Type, m.Value, *t.MinValue)
		} else if t.MaxValue != nil && m.Value > *t.MaxValue {
			alert = true
			msg = fmt.Sprintf("Cảnh báo: %s của bạn đang ở mức cao (%.2f), ngưỡng tối đa là %.2f", m.Type, m.Value, *t.MaxValue)
		}

		if alert && s.notificationSvc != nil {
			notification := notification.Notification{
				Title: "Cảnh báo sức khỏe",
				Body:  msg,
				Data: map[string]string{
					"type":        "health_alert",
					"metric_type": m.Type,
					"value":       fmt.Sprintf("%.2f", m.Value),
				},
			}
			err := s.notificationSvc.SendToUser(ctx, m.UserID, notification)
			if err != nil {
				log.Printf("Error sending notification to user %s: %v", m.UserID, err)
			}
		}
	}
}

func (s *serviceImpl) GetThresholds(ctx context.Context, userID string) ([]UserThreshold, error) {
	return s.repo.GetUserThresholds(ctx, userID)
}

func (s *serviceImpl) SetUserThreshold(ctx context.Context, threshold UserThreshold) error {
	return s.repo.UpsertThreshold(ctx, &threshold)
}

func (s *serviceImpl) GetChartData(ctx context.Context, userID, metricType, timeRange string, start, end *time.Time) ([]MetricDataPoint, error) {
	var startTime, endTime time.Time

	switch timeRange {
	case RangeLast24Hours:
		endTime = time.Now()
		startTime = endTime.Add(-24 * time.Hour)
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
	if days <= 1 {
		bucketSize = "1 hour" // or "15 minutes" based on preference, using 1 hour for now to ensure consistency, but if we have high res, 15m is good
		// Let's use 15 minutes for 24h
		bucketSize = "15 minutes"
	} else if days <= 2 {
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
