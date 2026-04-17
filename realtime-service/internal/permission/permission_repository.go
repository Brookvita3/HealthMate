package permission

import (
	"context"
)

type Repository interface {

	// CheckAccess checks if the observer can view the metric of the target user
	CheckAccess(ctx context.Context, observerUserID, targetUserID string, metricType string) (bool, error)

	// GetMetricWatchers returns a map of UserID -> []GroupID that authorized the user
	GetMetricWatchers(ctx context.Context, targetUserID, metricType string) (map[string][]string, error)
}
