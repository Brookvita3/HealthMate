package auth

import (
	"context"
	"testing"

	"healthmate/internal/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
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
	args := m.Called(ctx, params)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]user.User), args.Error(1)
}

func (m *MockUserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateAccessToken(user *user.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) GenerateRefreshToken(ctx context.Context, user *user.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidateToken(tokenString string) (map[string]any, error) {
	args := m.Called(tokenString)
	return args.Get(0).(map[string]any), args.Error(1)
}

func (m *MockTokenService) ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error) {
	args := m.Called(ctx, refreshToken)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
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

	t.Run("Register fail: User have account", func(t *testing.T) {

		mockRepo.On("GetUserByEmail", mock.Anything, "exists@example.com").Return(&user.User{
			Email: "exists@example.com",
		}, nil)

		createdUser, err := authService.RegisterWithEmail(context.Background(), "exists@example.com", "password123", "Existing User")

		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Equal(t, user.ErrEmailAlreadyRegistered, err)

		mockRepo.AssertExpectations(t)
	})

}

func TestLoginWithEmail(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockTokenService := new(MockTokenService)

	authService := NewAuthService(mockRepo, mockTokenService, "")

	t.Run("Login success ", func(t *testing.T) {

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &user.User{
			Email:    "test@example.com",
			Password: string(hashedPassword),
		}

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()

		mockTokenService.On("GenerateAccessToken", mock.AnythingOfType("*user.User")).Return("access_token", nil).Once()
		mockTokenService.On("GenerateRefreshToken", mock.Anything, mock.AnythingOfType("*user.User")).Return("refresh_token", nil).Once()

		result, err := authService.LoginWithEmail(context.Background(), "test@example.com", "password123")

		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, "access_token", result.AccessToken)
		assert.Equal(t, "refresh_token", result.RefreshToken)

		assert.NotNil(t, result.User)
		assert.Equal(t, mockUser.Email, result.User.Email)

		mockRepo.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
	})

	t.Run("Login failed: wrong password ", func(t *testing.T) {

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &user.User{
			Email:    "test@example.com",
			Password: string(hashedPassword),
		}

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()

		loginResult, err := authService.LoginWithEmail(context.Background(), "test@example.com", "wrong_password")

		assert.Error(t, err)
		assert.Equal(t, user.ErrInvalidCredentials, err)
		assert.Nil(t, loginResult)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Login failed: user not found ", func(t *testing.T) {

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()

		loginResult, err := authService.LoginWithEmail(context.Background(), "test@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, user.ErrInvalidCredentials, err)
		assert.Nil(t, loginResult)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Login failed: password not set (user login by google) ", func(t *testing.T) {

		mockUser := &user.User{
			Email: "test@example.com",
		}
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()

		loginResult, err := authService.LoginWithEmail(context.Background(), "test@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, user.ErrPasswordNotSet, err)
		assert.Nil(t, loginResult)

		mockRepo.AssertExpectations(t)
	})
}
