package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"healthmate/internal/common"
	"healthmate/internal/domain"
	"healthmate/internal/user"
	"healthmate/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func setupTest() (*mocks.Repository, *mocks.OTPService, *mocks.EmailService, *mocks.TokenService, Service) {
	mockRepo := new(mocks.Repository)
	mockTokenService := new(mocks.TokenService)
	mockOTPService := new(mocks.OTPService)
	mockEmailService := new(mocks.EmailService)
	authService := NewAuthService(mockRepo, mockTokenService, mockOTPService, mockEmailService, "")

	return mockRepo, mockOTPService, mockEmailService, mockTokenService, authService
}

func TestRegisterWithEmail(t *testing.T) {

	t.Run("Register success", func(t *testing.T) {

		mockRepo, mockOTPService, mockEmailService, _, authService := setupTest()

		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		mockOTPService.On("Generate", mock.Anything, "test@example.com").Return("123456", nil).Once()
		mockEmailService.On("SendOTP", mock.Anything, "test@example.com", "Test User", "123456").Return(nil).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.NoError(t, err)
		assert.NotNil(t, createdUser)
		assert.Equal(t, "test@example.com", createdUser.Email)
		assert.Equal(t, "Test User", createdUser.Name)
		assert.Equal(t, "unverified", createdUser.Status)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("Register fail: User already exists", func(t *testing.T) {

		mockRepo, _, _, _, authService := setupTest()

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

		mockRepo, _, _, _, authService := setupTest()

		dbError := errors.New("database connection lost")
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(dbError).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Equal(t, dbError, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Register fail: OTP generation fails", func(t *testing.T) {

		mockRepo, mockOTPService, _, _, authService := setupTest()

		otpError := errors.New("redis is down")
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		mockOTPService.On("Generate", mock.Anything, "test@example.com").Return("", otpError).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)
		assert.Nil(t, createdUser)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
	})

	t.Run("Register fail: Send OTP fails", func(t *testing.T) {

		mockRepo, mockOTPService, mockEmailService, _, authService := setupTest()

		mailError := errors.New("SMTP server down")
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		mockOTPService.On("Generate", mock.Anything, "test@example.com").Return("123456", nil).Once()
		mockEmailService.On("SendOTP", mock.Anything, "test@example.com", "Test User", "123456").Return(mailError).Once()

		createdUser, err := authService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)
		assert.Nil(t, createdUser)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

}

func TestLoginWithEmail(t *testing.T) {

	t.Run("Login success", func(t *testing.T) {

		mockRepo, _, _, mockTokenService, authService := setupTest()

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

		mockRepo, _, _, _, authService := setupTest()

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

		mockRepo, _, _, _, authService := setupTest()

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

		mockRepo, _, _, _, authService := setupTest()

		mockRepo.On("GetUserByEmail", mock.Anything, "notfound@example.com").Return(nil, user.ErrUserNotFound).Once()

		loginResult, err := authService.LoginWithEmail(context.Background(), "notfound@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, user.ErrInvalidCredentials, err)
		assert.Nil(t, loginResult)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Login failed: Password not set (e.g., social login)", func(t *testing.T) {

		mockRepo, _, _, _, authService := setupTest()

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

		mockRepo, _, _, mockTokenService, authService := setupTest()

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

func TestVerifyAccount(t *testing.T) {

	t.Run("Verify success", func(t *testing.T) {

		mockRepo, mockOTPService, mockEmailService, mockTokenService, authService := setupTest()

		mockUser := &domain.User{
			Id:     uuid.New(),
			Email:  "test@example.com",
			Name:   "Test User",
			Status: "unverified",
		}

		mockOTPService.On("Verify", mock.Anything, "test@example.com", "123456").Return(nil).Once()
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()

		expectedStatus := "verified"
		expectedParams := user.UpdateUserParams{Status: &expectedStatus}
		mockRepo.On("UpdateUser", mock.Anything, mockUser.Id, expectedParams).Return(nil).Once()

		mockTokenService.On("GenerateAccessToken", mock.AnythingOfType("*domain.User")).Return("access_token", nil).Once()
		mockTokenService.On("GenerateRefreshToken", mock.Anything, mock.AnythingOfType("*domain.User")).Return("refresh_token", nil).Once()

		mockOTPService.On("Delete", mock.Anything, "test@example.com").Return(nil).Once()
		mockEmailService.On("SendWelcomeEmail", mock.Anything, "test@example.com", "Test User").Return(nil).Once()

		result, err := authService.VerifyAccount(context.Background(), "test@example.com", "123456")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "access_token", result.AccessToken)
		assert.Equal(t, "refresh_token", result.RefreshToken)
		assert.Equal(t, "verified", result.User.Status)

		time.Sleep(50 * time.Millisecond)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("Verify failed: Invalid OTP", func(t *testing.T) {

		_, mockOTPService, _, _, authService := setupTest()

		otpError := errors.New("invalid OTP")
		mockOTPService.On("Verify", mock.Anything, "test@example.com", "wrong_otp").Return(otpError).Once()

		result, err := authService.VerifyAccount(context.Background(), "test@example.com", "wrong_otp")

		assert.Error(t, err)
		assert.Equal(t, otpError, err)
		assert.Nil(t, result)

		mockOTPService.AssertExpectations(t)
	})

	t.Run("Verify failed: Account already verified", func(t *testing.T) {

		mockRepo, mockOTPService, _, _, authService := setupTest()

		mockUser := &domain.User{
			Email:  "verified@example.com",
			Status: "verified",
		}
		mockOTPService.On("Verify", mock.Anything, "verified@example.com", "123456").Return(nil).Once()
		mockRepo.On("GetUserByEmail", mock.Anything, "verified@example.com").Return(mockUser, nil).Once()

		result, err := authService.VerifyAccount(context.Background(), "verified@example.com", "123456")

		assert.Error(t, err)
		assert.Equal(t, ErrAccountAlreadyVerified, err)
		assert.Nil(t, result)

		mockOTPService.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Verify failed: Update user status fails", func(t *testing.T) {

		mockRepo, mockOTPService, _, _, authService := setupTest()

		mockUser := &domain.User{Id: uuid.New(), Email: "test@example.com", Status: "unverified"}
		dbError := errors.New("database error")

		mockOTPService.On("Verify", mock.Anything, "test@example.com", "123456").Return(nil).Once()
		mockRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()
		mockRepo.On("UpdateUser", mock.Anything, mockUser.Id, mock.Anything).Return(dbError).Once()

		result, err := authService.VerifyAccount(context.Background(), "test@example.com", "123456")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)
		assert.Nil(t, result)

		mockOTPService.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}

func TestResendVerificationOTP(t *testing.T) {

	t.Run("Resend success", func(t *testing.T) {

		mockRepo, mockOTPService, mockEmailService, _, authService := setupTest()

		mockUser := &domain.User{
			Email:  "unverified@example.com",
			Name:   "Test User",
			Status: "unverified",
		}

		mockRepo.On("GetUserByEmail", mock.Anything, "unverified@example.com").Return(mockUser, nil).Once()
		mockOTPService.On("Generate", mock.Anything, "unverified@example.com").Return("654321", nil).Once()
		mockEmailService.On("ResendOTP", mock.Anything, "unverified@example.com", "Test User", "654321").Return(nil).Once()

		err := authService.ResendVerificationOTP(context.Background(), "unverified@example.com")

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("Resend for non-existent user returns no error", func(t *testing.T) {

		mockRepo, mockOTPService, mockEmailService, _, authService := setupTest()

		mockRepo.On("GetUserByEmail", mock.Anything, "notfound@example.com").Return(nil, user.ErrUserNotFound).Once()

		err := authService.ResendVerificationOTP(context.Background(), "notfound@example.com")

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertNotCalled(t, "Generate", mock.Anything, mock.Anything)
		mockEmailService.AssertNotCalled(t, "ResendOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Resend failed: Account is already verified", func(t *testing.T) {

		mockRepo, _, _, _, authService := setupTest()

		mockUser := &domain.User{
			Email:  "verified@example.com",
			Status: "verified", // Tài khoản đã xác thực
		}
		mockRepo.On("GetUserByEmail", mock.Anything, "verified@example.com").Return(mockUser, nil).Once()

		err := authService.ResendVerificationOTP(context.Background(), "verified@example.com")

		assert.Error(t, err)
		assert.Equal(t, ErrAccountAlreadyVerified, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Resend failed: OTP generation fails", func(t *testing.T) {

		mockRepo, mockOTPService, _, _, authService := setupTest()

		mockUser := &domain.User{
			Email:  "unverified@example.com",
			Status: "unverified",
		}
		otpError := errors.New("redis is down")

		mockRepo.On("GetUserByEmail", mock.Anything, "unverified@example.com").Return(mockUser, nil).Once()
		mockOTPService.On("Generate", mock.Anything, "unverified@example.com").Return("", otpError).Once()

		err := authService.ResendVerificationOTP(context.Background(), "unverified@example.com")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
	})

	t.Run("Resend failed: Sending email fails", func(t *testing.T) {

		mockRepo, mockOTPService, mockEmailService, _, authService := setupTest()

		mockUser := &domain.User{
			Email:  "unverified@example.com",
			Name:   "Test User",
			Status: "unverified",
		}
		mailError := errors.New("SMTP server is down")

		mockRepo.On("GetUserByEmail", mock.Anything, "unverified@example.com").Return(mockUser, nil).Once()
		mockOTPService.On("Generate", mock.Anything, "unverified@example.com").Return("654321", nil).Once()
		mockEmailService.On("ResendOTP", mock.Anything, "unverified@example.com", "Test User", "654321").Return(mailError).Once()

		err := authService.ResendVerificationOTP(context.Background(), "unverified@example.com")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)

		mockRepo.AssertExpectations(t)
		mockOTPService.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})
}

func TestSetPasswordForUser(t *testing.T) {

	t.Run("Set password success", func(t *testing.T) {

		mockRepo, _, _, _, authService := setupTest()

		userID := uuid.New()
		newPassword := "my-new-strong-password"

		mockRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil).Once()

		err := authService.SetPasswordForUser(context.Background(), userID, newPassword)

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Set password failed: Repository error", func(t *testing.T) {

		mockRepo, _, _, _, authService := setupTest()

		userID := uuid.New()
		dbError := errors.New("failed to update password in DB")

		mockRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(dbError).Once()

		err := authService.SetPasswordForUser(context.Background(), userID, "any-password")

		assert.Error(t, err)
		assert.Equal(t, dbError, err)

		mockRepo.AssertExpectations(t)
	})
}
