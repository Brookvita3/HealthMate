package auth

import (
	"context"
	"testing"

	"healthmate/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetUserById(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *user.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return m.Called(ctx, id, passwordHash).Error(0)
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, id uuid.UUID, params user.UpdateUserParams) error {
	return m.Called(ctx, id, params).Error(0)
}

func (m *MockUserRepository) ListUsers(ctx context.Context, params user.ListUsersParams) ([]user.User, error) {
	return m.Called(ctx, params).Get(0).([]user.User), m.Called(ctx, params).Error(1)
}

func (m *MockUserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}

func TestRegisterWithEmail(t *testing.T) {
	mockRepo := new(MockUserRepository)

	authService := NewAuthService(mockRepo, nil, "")

	t.Run("Register success ", func(t *testing.T) {

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.NoError(t, err)
		assert.NotNil(t, createdUser)
		assert.Equal(t, "test@example.com", createdUser.Email)
		assert.Equal(t, "Test User", createdUser.Name)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Register fail: User have App account", func(t *testing.T) {

		mockRepo.On("GetUserByEmail", mock.Anything, "exists@example.com").Return(&user.User{
			Email:    "exists@example.com",
			Provider: "HealthMate",
		}, nil)

		createdUser, err := authService.RegisterWithEmail(context.Background(), "exists@example.com", "password123", "Existing User")

		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Equal(t, user.ErrUserAlreadyExists, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Register fail: User have Google account", func(t *testing.T) {

		mockRepo.On("GetUserByEmail", mock.Anything, "google@example.com").Return(&user.User{
			Email:    "google@example.com",
			Provider: "Google",
		}, nil)

		createdUser, err := authService.RegisterWithEmail(context.Background(), "google@example.com", "password123", "Existing Google User")

		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Equal(t, user.ErrEmailAssociatedWithGoogle, err)
		mockRepo.AssertExpectations(t)
	})
}
