package permission

import (
	"auth-service/internal/domain"
	"context"

	"github.com/google/uuid"
)

// PermissionRepository defines the interface for managing metric sharing permissions.
type PermissionRepository interface {
	// SetPermission enables sharing of a specific metric type for a user in a group.
	// If sharedWithUserID is nil, it applies to all members (Group Rule).
	SetPermission(ctx context.Context, groupID, userID uuid.UUID, metricType string, sharedWithUserID *uuid.UUID) error

	// RevokePermission disables sharing of a specific metric type for a user in a group.
	RevokePermission(ctx context.Context, groupID, userID uuid.UUID, metricType string, sharedWithUserID *uuid.UUID) error

	// ListUserPermissionsInGroup retrieves all enabled sharing permissions for a user within a group.
	// If targetUserID is not nil, it filters for that specific member.
	ListUserPermissionsInGroup(ctx context.Context, groupID, userID uuid.UUID, targetUserID *uuid.UUID) ([]domain.Permission, error)

	// RevokeAllPermissions revokes ALL sharing permissions (global + member-specific) for a user in a group.
	// Prefer RevokeGlobalPermissions when only the global rule needs to be replaced.
	RevokeAllPermissions(ctx context.Context, groupID, userID uuid.UUID) error

	// RevokeGlobalPermissions revokes only the global rules (shared_with_user_id IS NULL) for a user.
	// Member-specific rules are intentionally preserved.
	RevokeGlobalPermissions(ctx context.Context, groupID, userID uuid.UUID) error

	// RevokeSpecificPermissions revokes all specific member permissions for a sharer in a group.
	RevokeSpecificPermissions(ctx context.Context, groupID, userID uuid.UUID, targetUserID uuid.UUID) error

	// ListAllMembersVisibleMetrics returns, for each member in the group (excluding viewerID),
	// the metric types that viewerID is permitted to see, applying the two-tier rule:
	// member has access_control row for viewer → use viewer-specific rules; otherwise → global rules.
	ListAllMembersVisibleMetrics(ctx context.Context, groupID, viewerID uuid.UUID) ([]domain.MemberVisibleMetrics, error)

	// IsMember checks if a user is an accepted member of a specific group.
	IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error)

	// IsMemberOrPendingApproval checks if a user is accepted or pending_owner_approval in a group.
	// Used for global permission updates so invited-but-not-yet-approved members can pre-set sharing.
	IsMemberOrPendingApproval(ctx context.Context, groupID, userID uuid.UUID) (bool, error)

	// IsValidMetricType checks if a metric type is defined in the database.
	IsValidMetricType(ctx context.Context, metricType string) (bool, error)

	// ListMetricTypes retrieves all available metric types in the system.
	ListMetricTypes(ctx context.Context) ([]domain.MetricType, error)
}
