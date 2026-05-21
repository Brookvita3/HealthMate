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

	// GetGroupMembersVisibleMetrics returns, for each member in the group (excluding viewerID),
	// the metric types the viewer is allowed to see based on each member's sharing settings.
	GetGroupMembersVisibleMetrics(ctx context.Context, groupID, viewerID uuid.UUID) ([]domain.MemberVisibleMetrics, error)

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

	if targetUserID == nil {
		// Global rule: pending_owner_approval members may pre-set their sharing.
		ok, err := s.repo.IsMemberOrPendingApproval(ctx, groupID, userID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrPermissionDenied
		}
	} else {
		return ErrUsePutForPerMemberSharing
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

	if targetUserID == nil {
		// Global rule: pending_owner_approval members may pre-set their sharing.
		ok, err := s.repo.IsMemberOrPendingApproval(ctx, groupID, userID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrPermissionDenied
		}
		// Update Global Rule — preserve member-specific rules.
		if err := s.repo.RevokeGlobalPermissions(ctx, groupID, userID); err != nil {
			return err
		}
	} else {
		// Member-specific rule: require fully accepted membership.
		isMember, err := s.repo.IsMember(ctx, groupID, userID)
		if err != nil {
			return err
		}
		if !isMember {
			return ErrPermissionDenied
		}

		if err := s.repo.RevokeSpecificPermissions(ctx, groupID, userID, *targetUserID); err != nil {
			return err
		}
		// Empty list: remove per-viewer filter; viewer falls back to sharer's global metrics.
		if len(metricTypes) == 0 {
			return nil
		}

		// Hierarchy check: member rules must be a subset of global rules.
		baseRules, err := s.repo.ListUserPermissionsInGroup(ctx, groupID, userID, nil)
		if err != nil {
			return err
		}
		allowedMetrics := make(map[string]bool)
		for _, b := range baseRules {
			if b.SharedWithUserId == nil && b.MetricType != "access_control" {
				allowedMetrics[b.MetricType] = true
			}
		}
		for _, m := range metricTypes {
			if !allowedMetrics[m] {
				return common.NewBusinessError(fmt.Sprintf("Metric '%s' is not allowed in this group rule (Base). Please enable it globally first.", m))
			}
		}
		if err := s.repo.SetPermission(ctx, groupID, userID, "access_control", targetUserID); err != nil {
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

// GetGroupMembersVisibleMetrics implementation.
func (s *serviceImpl) GetGroupMembersVisibleMetrics(ctx context.Context, groupID, viewerID uuid.UUID) ([]domain.MemberVisibleMetrics, error) {
	return s.repo.ListAllMembersVisibleMetrics(ctx, groupID, viewerID)
}

// ListMetricTypes implementation.
func (s *serviceImpl) ListMetricTypes(ctx context.Context) ([]domain.MetricType, error) {
	return s.repo.ListMetricTypes(ctx)
}
