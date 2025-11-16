package permission

import (
	"context"
)

type Repository interface {

	// CheckAccess checks if the observer can view the metric of the target user
	CheckAccess(ctx context.Context, observerUserID, targetUserID string, metricType string) (bool, error)
}
