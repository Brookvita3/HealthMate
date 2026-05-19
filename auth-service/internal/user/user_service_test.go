package user_test

import (
	"context"
	"errors"
	"testing"

	"auth-service/internal/common"
	"auth-service/internal/domain"
	"auth-service/internal/user"
	"auth-service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type userTestEnv struct {
	repo    *mocks.UserRepository
	service user.Service
}

func newUserTestEnv() *userTestEnv {
	repo := new(mocks.UserRepository)
	return &userTestEnv{
		repo:    repo,
		service: user.NewUserService(repo),
	}
}

// ─── GetUserProfile ───────────────────────────────────────────────────────────

func TestGetUserProfile(t *testing.T) {
	t.Run("success: returns user for valid ID", func(t *testing.T) {
		// Service must delegate directly to the repository without mutation.
		env := newUserTestEnv()
		id := uuid.New()
		want := &domain.User{Id: id, Email: "alice@example.com", Name: "Alice"}

		env.repo.On("GetUserById", mock.Anything, id).Return(want, nil).Once()

		got, err := env.service.GetUserProfile(context.Background(), id)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: user not found propagates ErrUserNotFound", func(t *testing.T) {
		// The service must surface repository errors unmodified.
		env := newUserTestEnv()
		id := uuid.New()

		env.repo.On("GetUserById", mock.Anything, id).Return(nil, user.ErrUserNotFound).Once()

		got, err := env.service.GetUserProfile(context.Background(), id)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, user.ErrUserNotFound)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: unexpected DB error is propagated", func(t *testing.T) {
		// Any non-domain error from the repo must also surface unchanged.
		env := newUserTestEnv()
		id := uuid.New()
		dbErr := errors.New("connection refused")

		env.repo.On("GetUserById", mock.Anything, id).Return(nil, dbErr).Once()

		_, err := env.service.GetUserProfile(context.Background(), id)

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── UpdateUserProfile ────────────────────────────────────────────────────────

func TestUpdateUserProfile(t *testing.T) {
	t.Run("success: partial update with only Name", func(t *testing.T) {
		// Service must accept params that contain at least one non-nil field.
		env := newUserTestEnv()
		id := uuid.New()
		name := "Bob"
		params := user.UpdateUserParams{Name: &name}

		env.repo.On("UpdateUser", mock.Anything, id, params).Return(nil).Once()

		err := env.service.UpdateUserProfile(context.Background(), id, params)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: update multiple fields at once", func(t *testing.T) {
		// Service should accept params with several non-nil fields.
		env := newUserTestEnv()
		id := uuid.New()
		phone := "+84900000000"
		tz := "Asia/Ho_Chi_Minh"
		params := user.UpdateUserParams{Phone: &phone, Timezone: &tz}

		env.repo.On("UpdateUser", mock.Anything, id, params).Return(nil).Once()

		err := env.service.UpdateUserProfile(context.Background(), id, params)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: all-nil params returns ErrInvalidRequest without hitting DB", func(t *testing.T) {
		// Guard clause must reject a completely empty update before touching the repo.
		env := newUserTestEnv()
		id := uuid.New()

		err := env.service.UpdateUserProfile(context.Background(), id, user.UpdateUserParams{})

		assert.ErrorIs(t, err, common.ErrInvalidRequest)
		env.repo.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("success: update allergies only", func(t *testing.T) {
		// Service must accept Allergies as the sole non-nil field.
		env := newUserTestEnv()
		id := uuid.New()
		allergies := []string{"Penicillin", "Aspirin"}
		params := user.UpdateUserParams{Allergies: &allergies}

		env.repo.On("UpdateUser", mock.Anything, id, mock.MatchedBy(func(p user.UpdateUserParams) bool {
			return p.Allergies != nil && len(*p.Allergies) == 2
		})).Return(nil).Once()

		err := env.service.UpdateUserProfile(context.Background(), id, params)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: allergies deduplicated case-insensitively", func(t *testing.T) {
		// Duplicates with different casing must be collapsed to one entry.
		env := newUserTestEnv()
		id := uuid.New()
		allergies := []string{"Penicillin", "penicillin", "ASPIRIN", "Aspirin"}
		params := user.UpdateUserParams{Allergies: &allergies}

		env.repo.On("UpdateUser", mock.Anything, id, mock.MatchedBy(func(p user.UpdateUserParams) bool {
			return p.Allergies != nil && len(*p.Allergies) == 2
		})).Return(nil).Once()

		err := env.service.UpdateUserProfile(context.Background(), id, params)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: empty allergies slice clears all allergies", func(t *testing.T) {
		// An empty (non-nil) slice must reach the repo to overwrite existing allergies.
		env := newUserTestEnv()
		id := uuid.New()
		allergies := []string{}
		params := user.UpdateUserParams{Allergies: &allergies}

		env.repo.On("UpdateUser", mock.Anything, id, mock.MatchedBy(func(p user.UpdateUserParams) bool {
			return p.Allergies != nil && len(*p.Allergies) == 0
		})).Return(nil).Once()

		err := env.service.UpdateUserProfile(context.Background(), id, params)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository failure is propagated", func(t *testing.T) {
		// DB errors from UpdateUser must bubble up unmodified.
		env := newUserTestEnv()
		id := uuid.New()
		name := "Carol"
		params := user.UpdateUserParams{Name: &name}
		dbErr := errors.New("deadlock detected")

		env.repo.On("UpdateUser", mock.Anything, id, params).Return(dbErr).Once()

		err := env.service.UpdateUserProfile(context.Background(), id, params)

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── ListUsers ────────────────────────────────────────────────────────────────

func TestListUsers(t *testing.T) {
	t.Run("success: returns users with valid params", func(t *testing.T) {
		// Default happy path: limit=20 is within bounds, passed unchanged.
		env := newUserTestEnv()
		params := user.ListUsersParams{Limit: 20, Offset: 0}
		want := []domain.User{
			{Id: uuid.New(), Email: "a@example.com"},
			{Id: uuid.New(), Email: "b@example.com"},
		}

		env.repo.On("ListUsers", mock.Anything, params).Return(want, nil).Once()

		got, err := env.service.ListUsers(context.Background(), params)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("guard: limit above 100 is capped to 100", func(t *testing.T) {
		// Service must not allow a caller to retrieve more than 100 users at once.
		env := newUserTestEnv()
		params := user.ListUsersParams{Limit: 500}
		expected := user.ListUsersParams{Limit: 100}

		env.repo.On("ListUsers", mock.Anything, expected).Return([]domain.User{}, nil).Once()

		_, err := env.service.ListUsers(context.Background(), params)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("guard: zero limit defaults to 10", func(t *testing.T) {
		// A caller that omits the limit should get a sensible default page size.
		env := newUserTestEnv()
		params := user.ListUsersParams{Limit: 0}
		expected := user.ListUsersParams{Limit: 10}

		env.repo.On("ListUsers", mock.Anything, expected).Return([]domain.User{}, nil).Once()

		_, err := env.service.ListUsers(context.Background(), params)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("guard: negative limit defaults to 10", func(t *testing.T) {
		// Negative limits are treated the same as zero.
		env := newUserTestEnv()
		params := user.ListUsersParams{Limit: -1}
		expected := user.ListUsersParams{Limit: 10}

		env.repo.On("ListUsers", mock.Anything, expected).Return([]domain.User{}, nil).Once()

		_, err := env.service.ListUsers(context.Background(), params)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: empty result set is not an error", func(t *testing.T) {
		// The service must return an empty slice, not an error, when nothing matches.
		env := newUserTestEnv()
		params := user.ListUsersParams{Limit: 10, Search: "no-match"}

		env.repo.On("ListUsers", mock.Anything, params).Return([]domain.User{}, nil).Once()

		got, err := env.service.ListUsers(context.Background(), params)

		assert.NoError(t, err)
		assert.Empty(t, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository failure is propagated", func(t *testing.T) {
		// DB errors from ListUsers must bubble up unmodified.
		env := newUserTestEnv()
		params := user.ListUsersParams{Limit: 10}
		dbErr := errors.New("timeout")

		env.repo.On("ListUsers", mock.Anything, params).Return(nil, dbErr).Once()

		got, err := env.service.ListUsers(context.Background(), params)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}
