package permission

import (
	"auth-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

// PermissionRepository defines the interface for managing metric sharing permissions.
type PermissionRepository interface {
	// SetPermission enables sharing of a specific metric type for a user in a group.
	SetPermission(ctx context.Context, groupID, userID uuid.UUID, metricType string) error

	// RevokePermission disables sharing of a specific metric type for a user in a group.
	RevokePermission(ctx context.Context, groupID, userID uuid.UUID, metricType string) error

	// ListUserPermissionsInGroup retrieves all enabled sharing permissions for a user within a group.
	ListUserPermissionsInGroup(ctx context.Context, groupID, userID uuid.UUID) ([]domain.Permission, error)
}
