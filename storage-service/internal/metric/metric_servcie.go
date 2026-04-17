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
	GetChartData(ctx context.Context, observerID, userID, metricType, timeRange string, start, end *time.Time) ([]MetricDataPoint, error)
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

	// Check thresholds for all recorded metrics
	go s.checkThresholds(context.Background(), metric)

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
		log.Printf("[Threshold] Error retrieving threshold for user %s, metric %s: %v", m.UserID, m.Type, err)
		return
	}

	if t.IsEnabled {
		alert := false
		level := ""
		thresholdVal := 0.0

		if t.MinValue != nil && m.Value < *t.MinValue {
			alert = true
			level = "low"
			thresholdVal = *t.MinValue
		} else if t.MaxValue != nil && m.Value > *t.MaxValue {
			alert = true
			level = "high"
			thresholdVal = *t.MaxValue
		}

		if alert {
			log.Printf("[Threshold Alert] User: %s, Metric: %s, Value: %.2f, Min: %v, Max: %v",
				m.UserID, m.Type, m.Value, t.MinValue, t.MaxValue)

			if s.notificationSvc == nil {
				log.Printf("[FCM] Notification service is not initialized, alert not sent.")
				return
			}

			// 1. Notify the owner (English)
			ownerMsg := fmt.Sprintf("Health Alert: Your %s is %s (%.2f). Threshold is %.2f", m.Type, level, m.Value, thresholdVal)
			ownerNotif := notification.Notification{
				Title: "Health Alert",
				Body:  ownerMsg,
				Data: map[string]string{
					"type":        "health_alert",
					"metric_type": m.Type,
					"value":       fmt.Sprintf("%.2f", m.Value),
					"level":       level,
				},
			}
			log.Printf("[FCM] Sending health alert to owner %s...", m.UserID)
			if err := s.notificationSvc.SendToUser(ctx, m.UserID, ownerNotif); err != nil {
				log.Printf("[FCM] Error sending to owner %s: %v", m.UserID, err)
			}

			// 2. Notify watchers (English)
			ownerName, err := s.repo.GetUserName(ctx, m.UserID)
			if err != nil {
				log.Printf("[Threshold] Failed to get owner name for %s: %v", m.UserID, err)
				ownerName = "A user"
			}

			watchers, err := s.repo.GetMetricWatchers(ctx, m.UserID, m.Type)
			if err != nil {
				log.Printf("[Threshold] Failed to get watchers for user %s: %v", m.UserID, err)
				return
			}

			for _, w := range watchers {
				watcherMsg := fmt.Sprintf("Health Alert: %s's %s in group %s is %s (%.2f). Threshold is %.2f",
					ownerName, m.Type, w.GroupName, level, m.Value, thresholdVal)

				watcherNotif := notification.Notification{
					Title: "Health Alert (Shared)",
					Body:  watcherMsg,
					Data: map[string]string{
						"type":        "health_alert_shared",
						"metric_type": m.Type,
						"value":       fmt.Sprintf("%.2f", m.Value),
						"owner_id":    m.UserID,
						"owner_name":  ownerName,
						"group_name":  w.GroupName,
						"level":       level,
					},
				}
				log.Printf("[FCM] Sending health alert to watcher %s (%s) for group %s...", w.UserID, w.UserName, w.GroupName)
				if err := s.notificationSvc.SendToUser(ctx, w.UserID, watcherNotif); err != nil {
					log.Printf("[FCM] Error sending to watcher %s: %v", w.UserID, err)
				}
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

func (s *serviceImpl) GetChartData(ctx context.Context, observerID, userID, metricType, timeRange string, start, end *time.Time) ([]MetricDataPoint, error) {
	// Check access permissions
	hasAccess, err := s.repo.CheckAccess(ctx, observerID, userID, metricType)
	if err != nil {
		return nil, fmt.Errorf("permission check failed: %w", err)
	}
	if !hasAccess {
		return nil, ErrPermissionDenied
	}

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
