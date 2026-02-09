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

	// UpdateSharing updates multiple sharing permissions for a user in a group.
	UpdateSharing(ctx context.Context, groupID, userID uuid.UUID, metricTypes []string) error

	// ListMetricTypes retrieves all available metric types in the system.
	ListMetricTypes(ctx context.Context) ([]domain.MetricType, error)
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
	isValid, err := s.repo.IsValidMetricType(ctx, metricType)
	if err != nil {
		return err
	}
	if !isValid {
		return ErrInvalidMetricType
	}

	// Double check membership to avoid FK violation
	isMember, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrPermissionDenied
	}

	return s.repo.SetPermission(ctx, groupID, userID, metricType)
}

// DisableSharing implements Service.DisableSharing.
// Revokes the sharing permission for a specific metric.
func (s *serviceImpl) DisableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string) error {
	isValid, err := s.repo.IsValidMetricType(ctx, metricType)
	if err != nil {
		return err
	}
	if !isValid {
		return ErrInvalidMetricType
	}
	return s.repo.RevokePermission(ctx, groupID, userID, metricType)
}

// GetPermissions implements Service.GetPermissions.
// Lists all allowed sharing permissions for a specific user in a group.
func (s *serviceImpl) GetPermissions(ctx context.Context, groupID, userID uuid.UUID) ([]domain.Permission, error) {
	return s.repo.ListUserPermissionsInGroup(ctx, groupID, userID)
}

// UpdateSharing implements Service.UpdateSharing.
// It revokes all current permissions and sets new ones.
func (s *serviceImpl) UpdateSharing(ctx context.Context, groupID, userID uuid.UUID, metricTypes []string) error {
	// Validate all metric types first
	for _, m := range metricTypes {
		isValid, err := s.repo.IsValidMetricType(ctx, m)
		if err != nil {
			return err
		}
		if !isValid {
			return ErrInvalidMetricType
		}
	}

	// Check if user is a member of the group before setting permissions
	// This prevents the foreign key constraint violation (SQLSTATE 23503)
	isMember, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrPermissionDenied
	}

	// For simplicity, revoke all and set new ones.
	// In a production environment, you might want to do this in a transaction.
	if err := s.repo.RevokeAllPermissions(ctx, groupID, userID); err != nil {
		return err
	}

	for _, m := range metricTypes {
		if err := s.repo.SetPermission(ctx, groupID, userID, m); err != nil {
			return err
		}
	}

	return nil
}

// ListMetricTypes implementation.
func (s *serviceImpl) ListMetricTypes(ctx context.Context) ([]domain.MetricType, error) {
	return s.repo.ListMetricTypes(ctx)
}
