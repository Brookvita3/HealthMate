package permission

import (
	"auth-service/internal/common"
	"auth-service/internal/domain"
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service defines the business logic for managing health metric sharing permissions.
type Service interface {
	// EnableSharing grants permission to share a specific metric type within a group.
	// If targetUserID is nil, it sets a Group Rule.
	EnableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string, targetUserID *uuid.UUID) error

	// DisableSharing revokes permission to share a specific metric type within a group.
	DisableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string, targetUserID *uuid.UUID) error

	// GetPermissions retrieves all shared metric types for a user in a group.
	// If targetUserID is not nil, it filters for that specific member.
	GetPermissions(ctx context.Context, groupID, userID uuid.UUID, targetUserID *uuid.UUID) ([]domain.Permission, error)

	// UpdateSharing updates multiple sharing permissions for a user in a group.
	// If targetUserID is not nil, it validates that all metricTypes are enabled in the Global Rule.
	UpdateSharing(ctx context.Context, groupID, userID uuid.UUID, targetUserID *uuid.UUID, metricTypes []string) error

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
func (s *serviceImpl) EnableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string, targetUserID *uuid.UUID) error {
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

	// Hierarchy check: If targetUserID is not nil, check if metricType is in Global Rules
	if targetUserID != nil {
		globalPerms, err := s.repo.ListUserPermissionsInGroup(ctx, groupID, userID, nil)
		if err != nil {
			return err
		}
		found := false
		for _, p := range globalPerms {
			if p.MetricType == metricType && p.SharedWithUserId == nil {
				found = true
				break
			}
		}
		if !found {
			return ErrPermissionDenied // Or a more specific error like "Metric not enabled in group level"
		}
	}

	return s.repo.SetPermission(ctx, groupID, userID, metricType, targetUserID)
}

// DisableSharing implements Service.DisableSharing.
// Revokes the sharing permission for a specific metric.
func (s *serviceImpl) DisableSharing(ctx context.Context, groupID, userID uuid.UUID, metricType string, targetUserID *uuid.UUID) error {
	isValid, err := s.repo.IsValidMetricType(ctx, metricType)
	if err != nil {
		return err
	}
	if !isValid {
		return ErrInvalidMetricType
	}
	return s.repo.RevokePermission(ctx, groupID, userID, metricType, targetUserID)
}

// GetPermissions implements Service.GetPermissions.
// Lists all allowed sharing permissions for a specific user in a group.
func (s *serviceImpl) GetPermissions(ctx context.Context, groupID, userID uuid.UUID, targetUserID *uuid.UUID) ([]domain.Permission, error) {
	return s.repo.ListUserPermissionsInGroup(ctx, groupID, userID, targetUserID)
}

// UpdateSharing implements Service.UpdateSharing.
// It revokes all current permissions and sets new ones.
func (s *serviceImpl) UpdateSharing(ctx context.Context, groupID, userID uuid.UUID, targetUserID *uuid.UUID, metricTypes []string) error {
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
	isMember, err := s.repo.IsMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrPermissionDenied
	}

	// Hierarchy check: Member rules must be subset of Global rules (Base)
	if targetUserID != nil {
		baseRules, err := s.repo.ListUserPermissionsInGroup(ctx, groupID, userID, nil)
		if err != nil {
			return err
		}

		allowedMetrics := make(map[string]bool)
		for _, b := range baseRules {
			allowedMetrics[b.MetricType] = true
		}

		for _, m := range metricTypes {
			if !allowedMetrics[m] {
				return common.NewBusinessError(fmt.Sprintf("Metric '%s' is not allowed in this group rule (Base). Please enable it globally first.", m))
			}
		}

		// Revoke all existing specific rules for this member before updating
		if err := s.repo.RevokeSpecificPermissions(ctx, groupID, userID, *targetUserID); err != nil {
			return err
		}

		// Always insert a special marker to signify this user is "Specially Managed" in this group.
		// This ensures they don't default back to Global defaults if their specific list is empty.
		if err := s.repo.SetPermission(ctx, groupID, userID, "access_control", targetUserID); err != nil {
			return err
		}
	} else {
		// Update Global Rule
		// Revoke all existing global rules first
		if err := s.repo.RevokeAllPermissions(ctx, groupID, userID); err != nil {
			return err
		}
	}

	for _, m := range metricTypes {
		if err := s.repo.SetPermission(ctx, groupID, userID, m, targetUserID); err != nil {
			return err
		}
	}

	return nil
}

// ListMetricTypes implementation.
func (s *serviceImpl) ListMetricTypes(ctx context.Context) ([]domain.MetricType, error) {
	return s.repo.ListMetricTypes(ctx)
}
