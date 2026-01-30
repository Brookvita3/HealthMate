package permission

import (
	"auth-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

// Service defines the business logic for managing health metric sharing permissions.
type Service interface {
	// EnableSharing grants permission to share a specific metric type within a group.
	EnableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string) error

	// DisableSharing revokes permission to share a specific metric type within a group.
	DisableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string) error

	// GetPermissions retrieves all shared metric types for a user in a group.
	GetPermissions(ctx context.Context, groupID, userID uuid.UUID) ([]domain.Permission, error)
}

type serviceImpl struct {
	repo PermissionRepository
}

func NewService(repo PermissionRepository) Service {
	return &serviceImpl{repo: repo}
}

// EnableSharing implements Service.EnableSharing.
// Validates the metric type before saving the permission.
func (s *serviceImpl) EnableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string) error {
	if !s.isValidMetricType(metricType) {
		return ErrInvalidMetricType
	}
	return s.repo.SetPermission(ctx, groupID, userID, metricType)
}

// DisableSharing implements Service.DisableSharing.
// Revokes the sharing permission for a specific metric.
func (s *serviceImpl) DisableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string) error {
	if !s.isValidMetricType(metricType) {
		return ErrInvalidMetricType
	}
	return s.repo.RevokePermission(ctx, groupID, userID, metricType)
}

// GetPermissions implements Service.GetPermissions.
// Lists all allowed sharing permissions for a specific user in a group.
func (s *serviceImpl) GetPermissions(ctx context.Context, groupID, userID uuid.UUID) ([]domain.Permission, error) {
	return s.repo.ListUserPermissionsInGroup(ctx, groupID, userID)
}

// isValidMetricType is a helper to check if the metric type is supported by the system.
func (s *serviceImpl) isValidMetricType(m string) bool {
	validTypes := map[string]bool{
		"heart_rate":      true,
		"steps_count":     true,
		"calories_burned": true,
	}
	return validTypes[m]
}
