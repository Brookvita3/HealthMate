package member_test

import (
	"context"
	"errors"
	"testing"

	"auth-service/internal/domain"
	"auth-service/internal/member"
	"auth-service/internal/user"
	"auth-service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockMemberRepo is a hand-written testify mock that satisfies the current
// MemberRepository interface.  The generated mock in mocks/ is missing the
// GetPendingApprovals method, so we define a correct implementation here
// without touching the existing generated file.
type mockMemberRepo struct{ mock.Mock }

func (m *mockMemberRepo) AddMember(ctx context.Context, groupID, userID, invitedBy uuid.UUID, role string) error {
	return m.Called(ctx, groupID, userID, invitedBy, role).Error(0)
}
func (m *mockMemberRepo) UpdateMemberStatus(ctx context.Context, groupID, userID uuid.UUID, status string) error {
	return m.Called(ctx, groupID, userID, status).Error(0)
}
func (m *mockMemberRepo) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	return m.Called(ctx, groupID, userID).Error(0)
}
func (m *mockMemberRepo) GetMember(ctx context.Context, groupID, userID uuid.UUID) (*domain.GroupMember, error) {
	ret := m.Called(ctx, groupID, userID)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(*domain.GroupMember), ret.Error(1)
}
func (m *mockMemberRepo) ListGroupMembers(ctx context.Context, groupID uuid.UUID) ([]domain.GroupMember, error) {
	ret := m.Called(ctx, groupID)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]domain.GroupMember), ret.Error(1)
}
func (m *mockMemberRepo) GetUserInvitations(ctx context.Context, userID uuid.UUID) ([]domain.InvitationResponse, error) {
	ret := m.Called(ctx, userID)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]domain.InvitationResponse), ret.Error(1)
}
func (m *mockMemberRepo) GroupExists(ctx context.Context, groupID uuid.UUID) (bool, error) {
	ret := m.Called(ctx, groupID)
	return ret.Bool(0), ret.Error(1)
}
func (m *mockMemberRepo) IsOwner(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	ret := m.Called(ctx, groupID, userID)
	return ret.Bool(0), ret.Error(1)
}
func (m *mockMemberRepo) GetGroupInvitations(ctx context.Context, groupID uuid.UUID) ([]domain.SentInvitationResponse, error) {
	ret := m.Called(ctx, groupID)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]domain.SentInvitationResponse), ret.Error(1)
}
func (m *mockMemberRepo) GetPendingApprovals(ctx context.Context, groupID uuid.UUID) ([]domain.SentInvitationResponse, error) {
	ret := m.Called(ctx, groupID)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]domain.SentInvitationResponse), ret.Error(1)
}
func (m *mockMemberRepo) CountMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	ret := m.Called(ctx, groupID)
	return ret.Int(0), ret.Error(1)
}

// ─── test env ────────────────────────────────────────────────────────────────

type memberTestEnv struct {
	repo     *mockMemberRepo
	userRepo *mocks.UserRepository
	service  member.Service
}

func newMemberEnv() *memberTestEnv {
	r := new(mockMemberRepo)
	u := new(mocks.UserRepository)
	return &memberTestEnv{repo: r, userRepo: u, service: member.NewService(r, u)}
}

// ─── InviteMember ─────────────────────────────────────────────────────────────

func TestInviteMember(t *testing.T) {
	t.Run("success: adds new member to existing group", func(t *testing.T) {
		// Happy path: group exists, invitee is not already a member.
		env := newMemberEnv()
		gID, uID, invBy := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, uID).Return(nil, errors.New("not found")).Once()
		env.repo.On("AddMember", mock.Anything, gID, uID, invBy, "member").Return(nil).Once()

		err := env.service.InviteMember(context.Background(), gID, uID, invBy)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: group does not exist", func(t *testing.T) {
		// GroupExists=false must return ErrInvalidGroup without attempting to add.
		env := newMemberEnv()
		gID, uID, invBy := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(false, nil).Once()

		err := env.service.InviteMember(context.Background(), gID, uID, invBy)

		assert.ErrorIs(t, err, member.ErrInvalidGroup)
		env.repo.AssertNotCalled(t, "AddMember", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("error: user is already a member", func(t *testing.T) {
		// GetMember returning a non-nil member means they already belong to the group.
		env := newMemberEnv()
		gID, uID, invBy := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, uID).Return(&domain.GroupMember{UserID: uID}, nil).Once()

		err := env.service.InviteMember(context.Background(), gID, uID, invBy)

		assert.ErrorIs(t, err, member.ErrMemberAlreadyExists)
	})

	t.Run("error: GroupExists repository failure propagates", func(t *testing.T) {
		env := newMemberEnv()
		gID, uID, invBy := uuid.New(), uuid.New(), uuid.New()
		dbErr := errors.New("db down")

		env.repo.On("GroupExists", mock.Anything, gID).Return(false, dbErr).Once()

		err := env.service.InviteMember(context.Background(), gID, uID, invBy)

		assert.ErrorIs(t, err, dbErr)
	})
}

// ─── InviteByEmail ────────────────────────────────────────────────────────────

func TestInviteByEmail(t *testing.T) {
	t.Run("success: finds user by email then invites them", func(t *testing.T) {
		// Full flow: email lookup → InviteMember internal call.
		env := newMemberEnv()
		gID, invBy := uuid.New(), uuid.New()
		foundUser := &domain.User{Id: uuid.New(), Email: "alice@example.com"}

		env.userRepo.On("GetUserByEmail", mock.Anything, "alice@example.com").Return(foundUser, nil).Once()
		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, foundUser.Id).Return(nil, errors.New("not found")).Once()
		env.repo.On("AddMember", mock.Anything, gID, foundUser.Id, invBy, "member").Return(nil).Once()

		err := env.service.InviteByEmail(context.Background(), gID, "alice@example.com", invBy)

		assert.NoError(t, err)
		env.userRepo.AssertExpectations(t)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: user with that email does not exist", func(t *testing.T) {
		// ErrUserNotFound from the user repo must surface unchanged.
		env := newMemberEnv()
		gID, invBy := uuid.New(), uuid.New()

		env.userRepo.On("GetUserByEmail", mock.Anything, "nobody@example.com").Return(nil, user.ErrUserNotFound).Once()

		err := env.service.InviteByEmail(context.Background(), gID, "nobody@example.com", invBy)

		assert.ErrorIs(t, err, user.ErrUserNotFound)
		env.repo.AssertNotCalled(t, "AddMember", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// ─── AcceptInvitation / RejectInvitation ──────────────────────────────────────

func TestAcceptInvitation(t *testing.T) {
	t.Run("success: pending invitation transitions to pending_owner_approval", func(t *testing.T) {
		// AcceptInvitation is a thin wrapper around UpdateMemberStatus.
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(&domain.GroupMember{Status: "pending"}, nil).Once()
		env.repo.On("UpdateMemberStatus", mock.Anything, gID, uID, "pending_owner_approval").Return(nil).Once()

		err := env.service.AcceptInvitation(context.Background(), gID, uID)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: member not found", func(t *testing.T) {
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(nil, member.ErrMemberNotFound).Once()

		err := env.service.AcceptInvitation(context.Background(), gID, uID)

		assert.ErrorIs(t, err, member.ErrMemberNotFound)
	})
}

func TestRejectInvitation(t *testing.T) {
	t.Run("success: pending invitation transitions to rejected", func(t *testing.T) {
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(&domain.GroupMember{Status: "pending"}, nil).Once()
		env.repo.On("UpdateMemberStatus", mock.Anything, gID, uID, "rejected").Return(nil).Once()

		err := env.service.RejectInvitation(context.Background(), gID, uID)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})
}

// ─── UpdateMemberStatus ───────────────────────────────────────────────────────

func TestUpdateMemberStatus(t *testing.T) {
	t.Run("success: pending → pending_owner_approval", func(t *testing.T) {
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(&domain.GroupMember{Status: "pending"}, nil).Once()
		env.repo.On("UpdateMemberStatus", mock.Anything, gID, uID, "pending_owner_approval").Return(nil).Once()

		err := env.service.UpdateMemberStatus(context.Background(), gID, uID, "pending_owner_approval")

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: pending → rejected", func(t *testing.T) {
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(&domain.GroupMember{Status: "pending"}, nil).Once()
		env.repo.On("UpdateMemberStatus", mock.Anything, gID, uID, "rejected").Return(nil).Once()

		err := env.service.UpdateMemberStatus(context.Background(), gID, uID, "rejected")

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("no-op: status already equals target", func(t *testing.T) {
		// If the member is already in the requested status the service must return
		// nil without calling UpdateMemberStatus.
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(&domain.GroupMember{Status: "rejected"}, nil).Once()

		err := env.service.UpdateMemberStatus(context.Background(), gID, uID, "rejected")

		assert.NoError(t, err)
		env.repo.AssertNotCalled(t, "UpdateMemberStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("error: only 'pending' members may respond to invitations", func(t *testing.T) {
		// A member in 'accepted' state cannot call UpdateMemberStatus.
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(&domain.GroupMember{Status: "accepted"}, nil).Once()

		err := env.service.UpdateMemberStatus(context.Background(), gID, uID, "pending_owner_approval")

		assert.ErrorIs(t, err, member.ErrInvalidStatus)
	})

	t.Run("error: target status must be pending_owner_approval or rejected", func(t *testing.T) {
		// Callers cannot set arbitrary status values via this method.
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(&domain.GroupMember{Status: "pending"}, nil).Once()

		err := env.service.UpdateMemberStatus(context.Background(), gID, uID, "accepted")

		assert.ErrorIs(t, err, member.ErrInvalidStatus)
	})

	t.Run("error: GetMember failure propagates", func(t *testing.T) {
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()
		dbErr := errors.New("db error")

		env.repo.On("GetMember", mock.Anything, gID, uID).Return(nil, dbErr).Once()

		err := env.service.UpdateMemberStatus(context.Background(), gID, uID, "rejected")

		assert.ErrorIs(t, err, dbErr)
	})
}

// ─── RemoveMember ─────────────────────────────────────────────────────────────

func TestRemoveMember(t *testing.T) {
	t.Run("success: owner removes a different member", func(t *testing.T) {
		env := newMemberEnv()
		gID, memberID, ownerID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()
		env.repo.On("RemoveMember", mock.Anything, gID, memberID).Return(nil).Once()

		err := env.service.RemoveMember(context.Background(), gID, memberID, ownerID)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: group does not exist", func(t *testing.T) {
		env := newMemberEnv()
		gID, memberID, ownerID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(false, nil).Once()

		err := env.service.RemoveMember(context.Background(), gID, memberID, ownerID)

		assert.ErrorIs(t, err, member.ErrInvalidGroup)
	})

	t.Run("error: requester is not the owner", func(t *testing.T) {
		// Non-owners cannot kick members.
		env := newMemberEnv()
		gID, memberID, nonOwnerID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, nonOwnerID).Return(false, nil).Once()

		err := env.service.RemoveMember(context.Background(), gID, memberID, nonOwnerID)

		assert.ErrorIs(t, err, member.ErrNotGroupOwner)
	})

	t.Run("error: owner cannot kick themselves", func(t *testing.T) {
		// userID == requesterID guard must trigger before RemoveMember is called.
		env := newMemberEnv()
		gID, ownerID := uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()

		err := env.service.RemoveMember(context.Background(), gID, ownerID, ownerID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot kick themselves")
		env.repo.AssertNotCalled(t, "RemoveMember", mock.Anything, mock.Anything, mock.Anything)
	})
}

// ─── LeaveGroup ───────────────────────────────────────────────────────────────

func TestLeaveGroup(t *testing.T) {
	t.Run("success: non-owner member leaves the group", func(t *testing.T) {
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, uID).Return(false, nil).Once()
		env.repo.On("RemoveMember", mock.Anything, gID, uID).Return(nil).Once()

		err := env.service.LeaveGroup(context.Background(), gID, uID)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: group not found", func(t *testing.T) {
		env := newMemberEnv()
		gID, uID := uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(false, nil).Once()

		err := env.service.LeaveGroup(context.Background(), gID, uID)

		assert.ErrorIs(t, err, member.ErrInvalidGroup)
	})

	t.Run("error: owner must transfer before leaving", func(t *testing.T) {
		// Group owner is blocked from leaving to prevent ownerless groups.
		env := newMemberEnv()
		gID, ownerID := uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()

		err := env.service.LeaveGroup(context.Background(), gID, ownerID)

		assert.ErrorIs(t, err, member.ErrOwnerCannotLeave)
		env.repo.AssertNotCalled(t, "RemoveMember", mock.Anything, mock.Anything, mock.Anything)
	})
}

// ─── GetMembers ───────────────────────────────────────────────────────────────

func TestGetMembers(t *testing.T) {
	t.Run("success: returns all members of an existing group", func(t *testing.T) {
		env := newMemberEnv()
		gID := uuid.New()
		want := []domain.GroupMember{
			{UserID: uuid.New(), Status: "accepted"},
			{UserID: uuid.New(), Status: "pending"},
		}

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("ListGroupMembers", mock.Anything, gID).Return(want, nil).Once()

		got, err := env.service.GetMembers(context.Background(), gID)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: group not found", func(t *testing.T) {
		env := newMemberEnv()
		gID := uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(false, nil).Once()

		got, err := env.service.GetMembers(context.Background(), gID)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, member.ErrInvalidGroup)
	})
}

// ─── GetUserInvitations ───────────────────────────────────────────────────────

func TestGetUserInvitations(t *testing.T) {
	t.Run("success: nil SharedMetrics slice is normalised to empty slice", func(t *testing.T) {
		// Callers expect a non-nil slice; the service must initialise nil fields.
		env := newMemberEnv()
		uID := uuid.New()
		raw := []domain.InvitationResponse{
			{SharedMetrics: nil},  // must be normalised
			{SharedMetrics: []string{"heart_rate"}},
		}

		env.repo.On("GetUserInvitations", mock.Anything, uID).Return(raw, nil).Once()

		got, err := env.service.GetUserInvitations(context.Background(), uID)

		assert.NoError(t, err)
		assert.NotNil(t, got[0].SharedMetrics)
		assert.Empty(t, got[0].SharedMetrics)
		assert.Equal(t, []string{"heart_rate"}, got[1].SharedMetrics)
	})

	t.Run("success: empty invitation list", func(t *testing.T) {
		env := newMemberEnv()
		uID := uuid.New()

		env.repo.On("GetUserInvitations", mock.Anything, uID).Return([]domain.InvitationResponse{}, nil).Once()

		got, err := env.service.GetUserInvitations(context.Background(), uID)

		assert.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		env := newMemberEnv()
		uID := uuid.New()
		dbErr := errors.New("query failed")

		env.repo.On("GetUserInvitations", mock.Anything, uID).Return(nil, dbErr).Once()

		_, err := env.service.GetUserInvitations(context.Background(), uID)

		assert.ErrorIs(t, err, dbErr)
	})
}

// ─── GetGroupInvitations ──────────────────────────────────────────────────────

func TestGetGroupInvitations(t *testing.T) {
	t.Run("success: owner sees all invitations", func(t *testing.T) {
		// The owner should receive the unfiltered invitation list.
		env := newMemberEnv()
		gID, ownerID, otherID := uuid.New(), uuid.New(), uuid.New()
		all := []domain.SentInvitationResponse{
			{InvitedBy: ownerID},
			{InvitedBy: otherID},
		}

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, ownerID).Return(&domain.GroupMember{Status: "accepted"}, nil).Once()
		env.repo.On("GetGroupInvitations", mock.Anything, gID).Return(all, nil).Once()

		got, err := env.service.GetGroupInvitations(context.Background(), gID, ownerID)

		assert.NoError(t, err)
		assert.Len(t, got, 2)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: non-owner member sees only their own invitations", func(t *testing.T) {
		// A regular member may only see invitations they personally sent.
		env := newMemberEnv()
		gID, memberID, otherID := uuid.New(), uuid.New(), uuid.New()
		all := []domain.SentInvitationResponse{
			{InvitedBy: memberID},
			{InvitedBy: otherID},
		}

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, memberID).Return(false, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, memberID).Return(&domain.GroupMember{Status: "accepted"}, nil).Once()
		env.repo.On("GetGroupInvitations", mock.Anything, gID).Return(all, nil).Once()

		got, err := env.service.GetGroupInvitations(context.Background(), gID, memberID)

		assert.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, memberID, got[0].InvitedBy)
	})

	t.Run("error: group not found", func(t *testing.T) {
		env := newMemberEnv()
		gID, rID := uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(false, nil).Once()

		_, err := env.service.GetGroupInvitations(context.Background(), gID, rID)

		assert.ErrorIs(t, err, member.ErrInvalidGroup)
	})

	t.Run("error: requester is not an accepted member", func(t *testing.T) {
		// A user with 'pending' status cannot view group invitations.
		env := newMemberEnv()
		gID, rID := uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, rID).Return(false, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, rID).Return(&domain.GroupMember{Status: "pending"}, nil).Once()

		_, err := env.service.GetGroupInvitations(context.Background(), gID, rID)

		assert.ErrorIs(t, err, member.ErrNotMember)
	})
}

// ─── ApproveJoinRequest ───────────────────────────────────────────────────────

func TestApproveJoinRequest(t *testing.T) {
	t.Run("success: owner approves a pending_owner_approval member", func(t *testing.T) {
		env := newMemberEnv()
		gID, ownerID, memberID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, memberID).Return(&domain.GroupMember{Status: "pending_owner_approval"}, nil).Once()
		env.repo.On("UpdateMemberStatus", mock.Anything, gID, memberID, "accepted").Return(nil).Once()

		err := env.service.ApproveJoinRequest(context.Background(), gID, ownerID, memberID)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: only the owner can approve join requests", func(t *testing.T) {
		env := newMemberEnv()
		gID, nonOwnerID, memberID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsOwner", mock.Anything, gID, nonOwnerID).Return(false, nil).Once()

		err := env.service.ApproveJoinRequest(context.Background(), gID, nonOwnerID, memberID)

		assert.ErrorIs(t, err, member.ErrNotGroupOwner)
	})

	t.Run("error: cannot approve a member not in pending_owner_approval state", func(t *testing.T) {
		// Trying to approve someone who is already 'accepted' must fail.
		env := newMemberEnv()
		gID, ownerID, memberID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, memberID).Return(&domain.GroupMember{Status: "accepted"}, nil).Once()

		err := env.service.ApproveJoinRequest(context.Background(), gID, ownerID, memberID)

		assert.ErrorIs(t, err, member.ErrNotPendingOwnerApproval)
	})
}

// ─── RejectJoinRequest ────────────────────────────────────────────────────────

func TestRejectJoinRequest(t *testing.T) {
	t.Run("success: owner rejects a pending_owner_approval member", func(t *testing.T) {
		env := newMemberEnv()
		gID, ownerID, memberID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, memberID).Return(&domain.GroupMember{Status: "pending_owner_approval"}, nil).Once()
		env.repo.On("UpdateMemberStatus", mock.Anything, gID, memberID, "rejected").Return(nil).Once()

		err := env.service.RejectJoinRequest(context.Background(), gID, ownerID, memberID)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: only owner can reject join requests", func(t *testing.T) {
		env := newMemberEnv()
		gID, nonOwnerID, memberID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsOwner", mock.Anything, gID, nonOwnerID).Return(false, nil).Once()

		err := env.service.RejectJoinRequest(context.Background(), gID, nonOwnerID, memberID)

		assert.ErrorIs(t, err, member.ErrNotGroupOwner)
	})

	t.Run("error: cannot reject member not in pending_owner_approval state", func(t *testing.T) {
		env := newMemberEnv()
		gID, ownerID, memberID := uuid.New(), uuid.New(), uuid.New()

		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()
		env.repo.On("GetMember", mock.Anything, gID, memberID).Return(&domain.GroupMember{Status: "rejected"}, nil).Once()

		err := env.service.RejectJoinRequest(context.Background(), gID, ownerID, memberID)

		assert.ErrorIs(t, err, member.ErrNotPendingOwnerApproval)
	})
}

// ─── GetPendingApprovals ──────────────────────────────────────────────────────

func TestGetPendingApprovals(t *testing.T) {
	t.Run("success: owner retrieves pending approval list", func(t *testing.T) {
		env := newMemberEnv()
		gID, ownerID := uuid.New(), uuid.New()
		want := []domain.SentInvitationResponse{
			{UserID: uuid.New(), Status: "pending_owner_approval"},
		}

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, ownerID).Return(true, nil).Once()
		env.repo.On("GetPendingApprovals", mock.Anything, gID).Return(want, nil).Once()

		got, err := env.service.GetPendingApprovals(context.Background(), gID, ownerID)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: group not found", func(t *testing.T) {
		env := newMemberEnv()
		gID, ownerID := uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(false, nil).Once()

		_, err := env.service.GetPendingApprovals(context.Background(), gID, ownerID)

		assert.ErrorIs(t, err, member.ErrInvalidGroup)
	})

	t.Run("error: non-owner cannot view pending approvals", func(t *testing.T) {
		env := newMemberEnv()
		gID, nonOwnerID := uuid.New(), uuid.New()

		env.repo.On("GroupExists", mock.Anything, gID).Return(true, nil).Once()
		env.repo.On("IsOwner", mock.Anything, gID, nonOwnerID).Return(false, nil).Once()

		_, err := env.service.GetPendingApprovals(context.Background(), gID, nonOwnerID)

		assert.ErrorIs(t, err, member.ErrNotGroupOwner)
	})
}
