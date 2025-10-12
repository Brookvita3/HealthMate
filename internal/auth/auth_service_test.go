package auth

import (
	"context"
	"errors"
	"testing"

	"healthmate/internal/common"
	"healthmate/internal/domain"
	"healthmate/internal/user"
	"healthmate/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestRegisterWithEmail(t *testing.T) {
	mockRepo := new(mocks.Repository)
	mockTokenService := new(mocks.TokenService)
	mockOTPService := new(mocks.OTPService)

	authService := NewAuthService(mockRepo, mockTokenService, mockOTPService, "")

	t.Run("Register success", func(t *testing.T) {

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil).Once()
		mockOTPService.On("Generate", mock.Anything, "test@example.com").Return("123456", nil).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.NoError(t, err)
		assert.NotNil(t, createdUser)
		assert.Equal(t, "test@example.com", createdUser.Email)
		assert.Equal(t, "Test User", createdUser.Name)
		assert.Equal(t, "unverified", createdUser.Status)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
	})

	t.Run("Register fail: User already exists", func(t *testing.T) {

		mockRepo.On("GetUserByEmail", mock.Anything, "exists@example.com").Return(&domain.User{
			Email: "exists@example.com",
		}, nil).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "exists@example.com", "password123", "Existing User")

		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Equal(t, user.ErrEmailAlreadyRegistered, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Register fail: CreateUser returns an error", func(t *testing.T) {

		dbError := errors.New("database connection lost")
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*user.User")).Return(dbError).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Equal(t, dbError, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Register fail: OTP generation fails", func(t *testing.T) {

		otpError := errors.New("redis is down")
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil).Once()
		mockOTPService.On("Generate", mock.Anything, "test@example.com").Return("", otpError).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)
		assert.Nil(t, createdUser)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
	})

}

func TestLoginWithEmail(t *testing.T) {

	mockRepo := new(mocks.Repository)
	mockTokenService := new(mocks.TokenService)
	mockOTPService := new(mocks.OTPService)

	authService := NewAuthService(mockRepo, mockTokenService, mockOTPService, "")

	t.Run("Login success", func(t *testing.T) {

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &domain.User{
			Email:    "test@example.com",
			Password: string(hashedPassword),
			Status:   "verified",
		}

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()
		mockTokenService.On("GenerateAccessToken", mockUser).Return("access_token", nil).Once()
		mockTokenService.On("GenerateRefreshToken", mock.Anything, mockUser).Return("refresh_token", nil).Once()

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

	t.Run("Login failed: Account not verified", func(t *testing.T) {

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &domain.User{
			Email:    "unverified@example.com",
			Password: string(hashedPassword),
			Status:   "unverified",
		}

		mockRepo.On("GetUserByEmail", mock.Anything, "unverified@example.com").Return(mockUser, nil).Once()

		loginResult, err := authService.LoginWithEmail(context.Background(), "unverified@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, ErrAccountNotVerified, err)
		assert.Nil(t, loginResult)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Login failed: Wrong password", func(t *testing.T) {

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &domain.User{
			Email:    "test@example.com",
			Password: string(hashedPassword),
			Status:   "verified",
		}

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()

		loginResult, err := authService.LoginWithEmail(context.Background(), "test@example.com", "wrong_password")

		assert.Error(t, err)
		assert.Equal(t, user.ErrInvalidCredentials, err)
		assert.Nil(t, loginResult)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Login failed: User not found", func(t *testing.T) {

		mockRepo.On("GetUserByEmail", mock.Anything, "notfound@example.com").Return(nil, user.ErrUserNotFound).Once()

		loginResult, err := authService.LoginWithEmail(context.Background(), "notfound@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, user.ErrInvalidCredentials, err)
		assert.Nil(t, loginResult)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Login failed: Password not set (e.g., social login)", func(t *testing.T) {

		mockUser := &domain.User{
			Email:  "social@example.com",
			Status: "verified",
		}
		mockRepo.On("GetUserByEmail", mock.Anything, "social@example.com").Return(mockUser, nil).Once()

		loginResult, err := authService.LoginWithEmail(context.Background(), "social@example.com", "any_password")

		assert.Error(t, err)
		assert.Equal(t, user.ErrPasswordNotSet, err)
		assert.Nil(t, loginResult)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Login failed: Generate access token error", func(t *testing.T) {

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &domain.User{
			Email:    "test@example.com",
			Password: string(hashedPassword),
			Status:   "verified",
		}
		tokenError := errors.New("failed to sign token")

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()
		mockTokenService.On("GenerateAccessToken", mockUser).Return("", tokenError).Once()

		result, err := authService.LoginWithEmail(context.Background(), "test@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, tokenError, err)
		assert.Nil(t, result)

		mockRepo.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
	})
}
