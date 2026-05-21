// Package permission_test covers group metric sharing rules:
// - Global share (targetUserID nil), including pending_owner_approval preset
// - Per-viewer share via UpdateSharing PUT only (access_control + metrics)
// - POST EnableSharing with targetUserID rejected (ErrUsePutForPerMemberSharing)
// - Empty metric_types + targetUserID resets viewer to global fallback
package permission_test

import (
	"context"
	"errors"
	"testing"

	"auth-service/internal/domain"
	"auth-service/internal/permission"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockPermRepo is a hand-written testify mock that satisfies the current
// PermissionRepository interface. The auto-generated mock in mocks/ has
// outdated signatures (missing targetUserID parameters), so we define a
// correct implementation here to keep tests compilable without touching
// existing generated files.
type mockPermRepo struct {
	mock.Mock
}

func (m *mockPermRepo) SetPermission(ctx context.Context, groupID, userID uuid.UUID, metricType string, sharedWithUserID *uuid.UUID) error {
	return m.Called(ctx, groupID, userID, metricType, sharedWithUserID).Error(0)
}

func (m *mockPermRepo) RevokePermission(ctx context.Context, groupID, userID uuid.UUID, metricType string, sharedWithUserID *uuid.UUID) error {
	return m.Called(ctx, groupID, userID, metricType, sharedWithUserID).Error(0)
}

func (m *mockPermRepo) RevokeAllPermissions(ctx context.Context, groupID, userID uuid.UUID) error {
	return m.Called(ctx, groupID, userID).Error(0)
}

func (m *mockPermRepo) RevokeGlobalPermissions(ctx context.Context, groupID, userID uuid.UUID) error {
	return m.Called(ctx, groupID, userID).Error(0)
}

func (m *mockPermRepo) ListAllMembersVisibleMetrics(ctx context.Context, groupID, viewerID uuid.UUID) ([]domain.MemberVisibleMetrics, error) {
	ret := m.Called(ctx, groupID, viewerID)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]domain.MemberVisibleMetrics), ret.Error(1)
}

func (m *mockPermRepo) RevokeSpecificPermissions(ctx context.Context, groupID, userID, targetUserID uuid.UUID) error {
	return m.Called(ctx, groupID, userID, targetUserID).Error(0)
}

func (m *mockPermRepo) ListUserPermissionsInGroup(ctx context.Context, groupID, userID uuid.UUID, targetUserID *uuid.UUID) ([]domain.Permission, error) {
	ret := m.Called(ctx, groupID, userID, targetUserID)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]domain.Permission), ret.Error(1)
}

func (m *mockPermRepo) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	ret := m.Called(ctx, groupID, userID)
	return ret.Bool(0), ret.Error(1)
}

func (m *mockPermRepo) IsMemberOrPendingApproval(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	ret := m.Called(ctx, groupID, userID)
	return ret.Bool(0), ret.Error(1)
}

func (m *mockPermRepo) IsValidMetricType(ctx context.Context, metricType string) (bool, error) {
	ret := m.Called(ctx, metricType)
	return ret.Bool(0), ret.Error(1)
}

func (m *mockPermRepo) ListMetricTypes(ctx context.Context) ([]domain.MetricType, error) {
	ret := m.Called(ctx)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]domain.MetricType), ret.Error(1)
}

// ptrUUID is a test helper that converts a UUID value to a pointer.
func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }

type permTestEnv struct {
	repo    *mockPermRepo
	service permission.Service
}

func newPermTestEnv() *permTestEnv {
	repo := new(mockPermRepo)
	return &permTestEnv{repo: repo, service: permission.NewService(repo)}
}

// ─── EnableSharing ────────────────────────────────────────────────────────────

func TestEnableSharing(t *testing.T) {
	t.Run("success: group-level rule (targetUserID nil)", func(t *testing.T) {
		// Global rule: IsMemberOrPendingApproval allows pending_owner_approval members too.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("IsMemberOrPendingApproval", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "heart_rate", (*uuid.UUID)(nil)).Return(nil).Once()

		err := env.service.EnableSharing(context.Background(), gID, uID, "heart_rate", nil)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: pending_owner_approval member can enable global sharing", func(t *testing.T) {
		// A member pending owner approval is allowed to set global sharing preferences.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "steps_count").Return(true, nil).Once()
		env.repo.On("IsMemberOrPendingApproval", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "steps_count", (*uuid.UUID)(nil)).Return(nil).Once()

		err := env.service.EnableSharing(context.Background(), gID, uID, "steps_count", nil)

		assert.NoError(t, err)
		env.repo.AssertNotCalled(t, "IsMember", mock.Anything, mock.Anything, mock.Anything)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: per-member sharing must use PUT not POST", func(t *testing.T) {
		env := newPermTestEnv()
		gID, uID, tID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()

		err := env.service.EnableSharing(context.Background(), gID, uID, "heart_rate", ptrUUID(tID))

		assert.ErrorIs(t, err, permission.ErrUsePutForPerMemberSharing)
		env.repo.AssertNotCalled(t, "SetPermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("error: invalid metric type is rejected before any DB call", func(t *testing.T) {
		// Unknown metric types must be rejected early; IsMember must not be called.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "unknown_metric").Return(false, nil).Once()

		err := env.service.EnableSharing(context.Background(), gID, uID, "unknown_metric", nil)

		assert.ErrorIs(t, err, permission.ErrInvalidMetricType)
		env.repo.AssertNotCalled(t, "IsMember", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("error: non-member cannot enable global sharing", func(t *testing.T) {
		// A user with no membership at all (not even pending) gets ErrPermissionDenied.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("IsMemberOrPendingApproval", mock.Anything, gID, uID).Return(false, nil).Once()

		err := env.service.EnableSharing(context.Background(), gID, uID, "heart_rate", nil)

		assert.ErrorIs(t, err, permission.ErrPermissionDenied)
		env.repo.AssertExpectations(t)
	})


	t.Run("error: IsValidMetricType repository failure propagates", func(t *testing.T) {
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()
		dbErr := errors.New("connection refused")

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(false, dbErr).Once()

		err := env.service.EnableSharing(context.Background(), gID, uID, "heart_rate", nil)

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── DisableSharing ───────────────────────────────────────────────────────────

func TestDisableSharing(t *testing.T) {
	t.Run("success: revokes group-level rule", func(t *testing.T) {
		// Happy path: valid metric type, revoke the group-level permission.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("RevokePermission", mock.Anything, gID, uID, "heart_rate", (*uuid.UUID)(nil)).Return(nil).Once()

		err := env.service.DisableSharing(context.Background(), gID, uID, "heart_rate", nil)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: revokes member-level rule", func(t *testing.T) {
		// When targetUserID is set, only that member's specific rule is revoked.
		env := newPermTestEnv()
		gID, uID, tID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("RevokePermission", mock.Anything, gID, uID, "heart_rate", ptrUUID(tID)).Return(nil).Once()

		err := env.service.DisableSharing(context.Background(), gID, uID, "heart_rate", ptrUUID(tID))

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: invalid metric type is rejected early", func(t *testing.T) {
		// RevokePermission must not be called for an unknown metric.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "bad_type").Return(false, nil).Once()

		err := env.service.DisableSharing(context.Background(), gID, uID, "bad_type", nil)

		assert.ErrorIs(t, err, permission.ErrInvalidMetricType)
		env.repo.AssertNotCalled(t, "RevokePermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("error: repository failure on revoke propagates", func(t *testing.T) {
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()
		dbErr := errors.New("timeout")

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("RevokePermission", mock.Anything, gID, uID, "heart_rate", (*uuid.UUID)(nil)).Return(dbErr).Once()

		err := env.service.DisableSharing(context.Background(), gID, uID, "heart_rate", nil)

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── GetPermissions ───────────────────────────────────────────────────────────

func TestGetPermissions(t *testing.T) {
	t.Run("success: returns all permissions for a user in a group", func(t *testing.T) {
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()
		want := []domain.Permission{
			{GroupId: gID, UserId: uID, MetricType: "heart_rate"},
			{GroupId: gID, UserId: uID, MetricType: "steps_count"},
		}

		env.repo.On("ListUserPermissionsInGroup", mock.Anything, gID, uID, (*uuid.UUID)(nil)).Return(want, nil).Once()

		got, err := env.service.GetPermissions(context.Background(), gID, uID, nil)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: filters by targetUserID when provided", func(t *testing.T) {
		// Service passes targetUserID straight through to the repo.
		env := newPermTestEnv()
		gID, uID, tID := uuid.New(), uuid.New(), uuid.New()
		want := []domain.Permission{
			{GroupId: gID, UserId: uID, MetricType: "heart_rate", SharedWithUserId: ptrUUID(tID)},
		}

		env.repo.On("ListUserPermissionsInGroup", mock.Anything, gID, uID, ptrUUID(tID)).Return(want, nil).Once()

		got, err := env.service.GetPermissions(context.Background(), gID, uID, ptrUUID(tID))

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()
		dbErr := errors.New("query failed")

		env.repo.On("ListUserPermissionsInGroup", mock.Anything, gID, uID, (*uuid.UUID)(nil)).Return(nil, dbErr).Once()

		got, err := env.service.GetPermissions(context.Background(), gID, uID, nil)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── UpdateSharing ────────────────────────────────────────────────────────────

func TestUpdateSharing(t *testing.T) {
	t.Run("success: replaces global rule without touching member-specific rules", func(t *testing.T) {
		// Global update uses IsMemberOrPendingApproval and RevokeGlobalPermissions.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()
		metrics := []string{"heart_rate", "steps_count"}

		for _, m := range metrics {
			env.repo.On("IsValidMetricType", mock.Anything, m).Return(true, nil).Once()
		}
		env.repo.On("IsMemberOrPendingApproval", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("RevokeGlobalPermissions", mock.Anything, gID, uID).Return(nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "heart_rate", (*uuid.UUID)(nil)).Return(nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "steps_count", (*uuid.UUID)(nil)).Return(nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, nil, metrics)

		assert.NoError(t, err)
		env.repo.AssertNotCalled(t, "RevokeAllPermissions", mock.Anything, mock.Anything, mock.Anything)
		env.repo.AssertNotCalled(t, "IsMember", mock.Anything, mock.Anything, mock.Anything)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: pending_owner_approval member can set global permissions", func(t *testing.T) {
		// Members pending owner approval can pre-set their global sharing before being fully accepted.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()
		metrics := []string{"heart_rate"}

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("IsMemberOrPendingApproval", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("RevokeGlobalPermissions", mock.Anything, gID, uID).Return(nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "heart_rate", (*uuid.UUID)(nil)).Return(nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, nil, metrics)

		assert.NoError(t, err)
		env.repo.AssertNotCalled(t, "IsMember", mock.Anything, mock.Anything, mock.Anything)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: replaces member rule when metrics are subset of global rule", func(t *testing.T) {
		// Member-level update: revoke specific rules, set access_control marker, then re-insert.
		env := newPermTestEnv()
		gID, uID, tID := uuid.New(), uuid.New(), uuid.New()
		metrics := []string{"heart_rate"}

		globalPerms := []domain.Permission{
			{GroupId: gID, UserId: uID, MetricType: "heart_rate"},
			{GroupId: gID, UserId: uID, MetricType: "steps_count"},
		}

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("IsMember", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("RevokeSpecificPermissions", mock.Anything, gID, uID, tID).Return(nil).Once()
		env.repo.On("ListUserPermissionsInGroup", mock.Anything, gID, uID, (*uuid.UUID)(nil)).Return(globalPerms, nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "access_control", ptrUUID(tID)).Return(nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "heart_rate", ptrUUID(tID)).Return(nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, ptrUUID(tID), metrics)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: empty member metrics resets to global fallback", func(t *testing.T) {
		env := newPermTestEnv()
		gID, uID, tID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsMember", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("RevokeSpecificPermissions", mock.Anything, gID, uID, tID).Return(nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, ptrUUID(tID), []string{})

		assert.NoError(t, err)
		env.repo.AssertNotCalled(t, "SetPermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: invalid metric type in list stops processing", func(t *testing.T) {
		// The service validates ALL metric types before touching the database.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "bad_metric").Return(false, nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, nil, []string{"bad_metric"})

		assert.ErrorIs(t, err, permission.ErrInvalidMetricType)
		env.repo.AssertNotCalled(t, "RevokeAllPermissions", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("error: non-member cannot update global sharing", func(t *testing.T) {
		// Users with no membership (not even pending) are denied even for global updates.
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("IsMemberOrPendingApproval", mock.Anything, gID, uID).Return(false, nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, nil, []string{"heart_rate"})

		assert.ErrorIs(t, err, permission.ErrPermissionDenied)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: member rule rejected when metric not in global rule", func(t *testing.T) {
		// A member-specific rule cannot include metrics absent from the group-level rule.
		env := newPermTestEnv()
		gID, uID, tID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsValidMetricType", mock.Anything, "calories").Return(true, nil).Once()
		env.repo.On("IsMember", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("RevokeSpecificPermissions", mock.Anything, gID, uID, tID).Return(nil).Once()
		env.repo.On("ListUserPermissionsInGroup", mock.Anything, gID, uID, (*uuid.UUID)(nil)).Return([]domain.Permission{
			{GroupId: gID, UserId: uID, MetricType: "heart_rate"},
		}, nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, ptrUUID(tID), []string{"calories"})

		assert.Error(t, err)
		env.repo.AssertNotCalled(t, "SetPermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: RevokeGlobalPermissions failure propagates", func(t *testing.T) {
		env := newPermTestEnv()
		gID, uID := uuid.New(), uuid.New()
		dbErr := errors.New("database error")

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("IsMemberOrPendingApproval", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("RevokeGlobalPermissions", mock.Anything, gID, uID).Return(dbErr).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, nil, []string{"heart_rate"})

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: member hierarchy check ignores specific rows and access_control", func(t *testing.T) {
		// access_control and member-specific rows must NOT count as allowed base metrics.
		// Only global rows (shared_with_user_id IS NULL, not access_control) are the base.
		env := newPermTestEnv()
		gID, uID, tID := uuid.New(), uuid.New(), uuid.New()
		tID2 := uuid.New()

		// Simulate: global rule has heart_rate only.
		// There is also an access_control specific row and a steps_count specific row for another member.
		// steps_count must NOT be considered allowed.
		mixedPerms := []domain.Permission{
			{GroupId: gID, UserId: uID, MetricType: "heart_rate", SharedWithUserId: nil},
			{GroupId: gID, UserId: uID, MetricType: "access_control", SharedWithUserId: ptrUUID(tID2)},
			{GroupId: gID, UserId: uID, MetricType: "steps_count", SharedWithUserId: ptrUUID(tID2)},
		}

		env.repo.On("IsValidMetricType", mock.Anything, "heart_rate").Return(true, nil).Once()
		env.repo.On("IsMember", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("RevokeSpecificPermissions", mock.Anything, gID, uID, tID).Return(nil).Once()
		env.repo.On("ListUserPermissionsInGroup", mock.Anything, gID, uID, (*uuid.UUID)(nil)).Return(mixedPerms, nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "access_control", ptrUUID(tID)).Return(nil).Once()
		env.repo.On("SetPermission", mock.Anything, gID, uID, "heart_rate", ptrUUID(tID)).Return(nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, ptrUUID(tID), []string{"heart_rate"})

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: member rule rejected when metric only in specific row but not global", func(t *testing.T) {
		// steps_count exists as a specific row for another member but NOT as a global rule.
		// It must still be rejected for a new member-level rule.
		env := newPermTestEnv()
		gID, uID, tID := uuid.New(), uuid.New(), uuid.New()
		tID2 := uuid.New()

		mixedPerms := []domain.Permission{
			{GroupId: gID, UserId: uID, MetricType: "heart_rate", SharedWithUserId: nil},
			{GroupId: gID, UserId: uID, MetricType: "steps_count", SharedWithUserId: ptrUUID(tID2)},
		}

		env.repo.On("IsValidMetricType", mock.Anything, "steps_count").Return(true, nil).Once()
		env.repo.On("IsMember", mock.Anything, gID, uID).Return(true, nil).Once()
		env.repo.On("RevokeSpecificPermissions", mock.Anything, gID, uID, tID).Return(nil).Once()
		env.repo.On("ListUserPermissionsInGroup", mock.Anything, gID, uID, (*uuid.UUID)(nil)).Return(mixedPerms, nil).Once()

		err := env.service.UpdateSharing(context.Background(), gID, uID, ptrUUID(tID), []string{"steps_count"})

		assert.Error(t, err)
		env.repo.AssertNotCalled(t, "SetPermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		env.repo.AssertExpectations(t)
	})
}

// ─── ListMetricTypes ──────────────────────────────────────────────────────────

func TestListMetricTypes(t *testing.T) {
	t.Run("success: returns all metric types from repository", func(t *testing.T) {
		env := newPermTestEnv()
		want := []domain.MetricType{
			{Name: "heart_rate"},
			{Name: "steps_count"},
			{Name: "calories"},
		}

		env.repo.On("ListMetricTypes", mock.Anything).Return(want, nil).Once()

		got, err := env.service.ListMetricTypes(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		env := newPermTestEnv()
		dbErr := errors.New("table not found")

		env.repo.On("ListMetricTypes", mock.Anything).Return(nil, dbErr).Once()

		got, err := env.service.ListMetricTypes(context.Background())

		assert.Nil(t, got)
		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}
