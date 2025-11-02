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

type testEnv struct {
	UserRepository *mocks.UserRepository
	OTPService     *mocks.OTPService
	EmailService   *mocks.EmailService
	TokenService   *mocks.TokenService
	GoogleVerifier *mocks.GoogleTokenVerifier
	AuthService    Service
}

func setupTest() *testEnv {
	mockUserRepository := new(mocks.UserRepository)
	mocktokenService := new(mocks.TokenService)
	mockOTPService := new(mocks.OTPService)
	mockEmailService := new(mocks.EmailService)
	mockGoogleVerifier := new(mocks.GoogleTokenVerifier)

	authService := NewAuthService(
		mockUserRepository,
		mocktokenService,
		mockOTPService,
		mockEmailService,
		mockGoogleVerifier,
	)

	return &testEnv{
		UserRepository: mockUserRepository,
		OTPService:     mockOTPService,
		EmailService:   mockEmailService,
		TokenService:   mocktokenService,
		GoogleVerifier: mockGoogleVerifier,
		AuthService:    authService,
	}
}
func TestRegisterWithEmail(t *testing.T) {

	t.Run("Register success", func(t *testing.T) {

		env := setupTest()

		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		env.UserRepository.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		env.OTPService.On("Generate", mock.Anything, "test@example.com").Return("123456", nil).Once()
		env.EmailService.On("SendOTP", mock.Anything, "test@example.com", "Test User", "123456").Return(nil).Once()

		createdUser, err := env.AuthService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.NoError(t, err)
		assert.NotNil(t, createdUser)
		assert.Equal(t, "test@example.com", createdUser.Email)
		assert.Equal(t, "Test User", createdUser.Name)
		assert.Equal(t, "unverified", createdUser.Status)

		env.UserRepository.AssertExpectations(t)
		env.OTPService.AssertExpectations(t)
		env.EmailService.AssertExpectations(t)
	})

	t.Run("Register fail: User already exists", func(t *testing.T) {

		env := setupTest()

		env.UserRepository.On("GetUserByEmail", mock.Anything, "exists@example.com").Return(&domain.User{
			Email: "exists@example.com",
		}, nil).Once()

		createdUser, err := env.AuthService.RegisterWithEmail(context.Background(), "exists@example.com", "password123", "Existing User")

		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Equal(t, user.ErrEmailAlreadyRegistered, err)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Register fail: CreateUser returns an error", func(t *testing.T) {

		env := setupTest()

		dbError := errors.New("database connection lost")
		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		env.UserRepository.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(dbError).Once()

		createdUser, err := env.AuthService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.Error(t, err)
		assert.Nil(t, createdUser)
		assert.Equal(t, dbError, err)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Register fail: OTP generation fails", func(t *testing.T) {

		env := setupTest()

		otpError := errors.New("redis is down")
		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		env.UserRepository.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		env.OTPService.On("Generate", mock.Anything, "test@example.com").Return("", otpError).Once()

		createdUser, err := env.AuthService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)
		assert.Nil(t, createdUser)

		env.UserRepository.AssertExpectations(t)
		env.OTPService.AssertExpectations(t)
	})

	t.Run("Register fail: Send OTP fails", func(t *testing.T) {

		env := setupTest()

		mailError := errors.New("SMTP server down")
		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, user.ErrUserNotFound).Once()
		env.UserRepository.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		env.OTPService.On("Generate", mock.Anything, "test@example.com").Return("123456", nil).Once()
		env.EmailService.On("SendOTP", mock.Anything, "test@example.com", "Test User", "123456").Return(mailError).Once()

		createdUser, err := env.AuthService.RegisterWithEmail(context.Background(), "test@example.com", "password123", "Test User")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)
		assert.Nil(t, createdUser)

		env.UserRepository.AssertExpectations(t)
		env.OTPService.AssertExpectations(t)
		env.EmailService.AssertExpectations(t)
	})

}

func TestLoginWithEmail(t *testing.T) {

	t.Run("Login success", func(t *testing.T) {

		env := setupTest()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &domain.User{
			Email:    "test@example.com",
			Password: string(hashedPassword),
			Status:   "verified",
		}

		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()
		env.TokenService.On("GenerateAccessToken", mockUser).Return("access_token", nil).Once()
		env.TokenService.On("GenerateRefreshToken", mock.Anything, mockUser).Return("refresh_token", nil).Once()

		result, err := env.AuthService.LoginWithEmail(context.Background(), "test@example.com", "password123")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "access_token", result.AccessToken)
		assert.Equal(t, "refresh_token", result.RefreshToken)
		assert.NotNil(t, result.User)
		assert.Equal(t, mockUser.Email, result.User.Email)

		env.UserRepository.AssertExpectations(t)
		env.TokenService.AssertExpectations(t)
	})

	t.Run("Login failed: Account not verified", func(t *testing.T) {

		env := setupTest()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &domain.User{
			Email:    "unverified@example.com",
			Password: string(hashedPassword),
			Status:   "unverified",
		}

		env.UserRepository.On("GetUserByEmail", mock.Anything, "unverified@example.com").Return(mockUser, nil).Once()

		loginResult, err := env.AuthService.LoginWithEmail(context.Background(), "unverified@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, ErrAccountNotVerified, err)
		assert.Nil(t, loginResult)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Login failed: Wrong password", func(t *testing.T) {

		env := setupTest()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &domain.User{
			Email:    "test@example.com",
			Password: string(hashedPassword),
			Status:   "verified",
		}

		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()

		loginResult, err := env.AuthService.LoginWithEmail(context.Background(), "test@example.com", "wrong_password")

		assert.Error(t, err)
		assert.Equal(t, user.ErrInvalidCredentials, err)
		assert.Nil(t, loginResult)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Login failed: User not found", func(t *testing.T) {

		env := setupTest()

		env.UserRepository.On("GetUserByEmail", mock.Anything, "notfound@example.com").Return(nil, user.ErrUserNotFound).Once()

		loginResult, err := env.AuthService.LoginWithEmail(context.Background(), "notfound@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, user.ErrInvalidCredentials, err)
		assert.Nil(t, loginResult)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Login failed: Password not set (e.g., social login)", func(t *testing.T) {

		env := setupTest()

		mockUser := &domain.User{
			Email:  "social@example.com",
			Status: "verified",
		}
		env.UserRepository.On("GetUserByEmail", mock.Anything, "social@example.com").Return(mockUser, nil).Once()

		loginResult, err := env.AuthService.LoginWithEmail(context.Background(), "social@example.com", "any_password")

		assert.Error(t, err)
		assert.Equal(t, user.ErrPasswordNotSet, err)
		assert.Nil(t, loginResult)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Login failed: Generate access token error", func(t *testing.T) {

		env := setupTest()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		mockUser := &domain.User{
			Email:    "test@example.com",
			Password: string(hashedPassword),
			Status:   "verified",
		}
		tokenError := errors.New("failed to sign token")

		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()
		env.TokenService.On("GenerateAccessToken", mockUser).Return("", tokenError).Once()

		result, err := env.AuthService.LoginWithEmail(context.Background(), "test@example.com", "password123")

		assert.Error(t, err)
		assert.Equal(t, tokenError, err)
		assert.Nil(t, result)

		env.UserRepository.AssertExpectations(t)
		env.TokenService.AssertExpectations(t)
	})
}

func TestVerifyAccount(t *testing.T) {

	t.Run("Verify success", func(t *testing.T) {

		env := setupTest()

		mockUser := &domain.User{
			Id:     uuid.New(),
			Email:  "test@example.com",
			Name:   "Test User",
			Status: "unverified",
		}

		env.OTPService.On("Verify", mock.Anything, "test@example.com", "123456").Return(nil).Once()
		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()

		expectedStatus := "verified"
		expectedParams := user.UpdateUserParams{Status: &expectedStatus}
		env.UserRepository.On("UpdateUser", mock.Anything, mockUser.Id, expectedParams).Return(nil).Once()

		env.TokenService.On("GenerateAccessToken", mock.AnythingOfType("*domain.User")).Return("access_token", nil).Once()
		env.TokenService.On("GenerateRefreshToken", mock.Anything, mock.AnythingOfType("*domain.User")).Return("refresh_token", nil).Once()

		env.OTPService.On("Delete", mock.Anything, "test@example.com").Return(nil).Once()
		env.EmailService.On("SendWelcomeEmail", mock.Anything, "test@example.com", "Test User").Return(nil).Once()

		result, err := env.AuthService.VerifyAccount(context.Background(), "test@example.com", "123456")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "access_token", result.AccessToken)
		assert.Equal(t, "refresh_token", result.RefreshToken)
		assert.Equal(t, "verified", result.User.Status)

		time.Sleep(50 * time.Millisecond)

		env.UserRepository.AssertExpectations(t)
		env.OTPService.AssertExpectations(t)
		env.TokenService.AssertExpectations(t)
		env.EmailService.AssertExpectations(t)
	})

	t.Run("Verify failed: Invalid OTP", func(t *testing.T) {

		env := setupTest()

		otpError := errors.New("invalid OTP")
		env.OTPService.On("Verify", mock.Anything, "test@example.com", "wrong_otp").Return(otpError).Once()

		result, err := env.AuthService.VerifyAccount(context.Background(), "test@example.com", "wrong_otp")

		assert.Error(t, err)
		assert.Equal(t, otpError, err)
		assert.Nil(t, result)

		env.OTPService.AssertExpectations(t)
	})

	t.Run("Verify failed: Account already verified", func(t *testing.T) {

		env := setupTest()

		mockUser := &domain.User{
			Email:  "verified@example.com",
			Status: "verified",
		}
		env.OTPService.On("Verify", mock.Anything, "verified@example.com", "123456").Return(nil).Once()
		env.UserRepository.On("GetUserByEmail", mock.Anything, "verified@example.com").Return(mockUser, nil).Once()

		result, err := env.AuthService.VerifyAccount(context.Background(), "verified@example.com", "123456")

		assert.Error(t, err)
		assert.Equal(t, ErrAccountAlreadyVerified, err)
		assert.Nil(t, result)

		env.OTPService.AssertExpectations(t)
		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Verify failed: Update user status fails", func(t *testing.T) {

		env := setupTest()

		mockUser := &domain.User{Id: uuid.New(), Email: "test@example.com", Status: "unverified"}
		dbError := errors.New("database error")

		env.OTPService.On("Verify", mock.Anything, "test@example.com", "123456").Return(nil).Once()
		env.UserRepository.On("GetUserByEmail", mock.Anything, "test@example.com").Return(mockUser, nil).Once()
		env.UserRepository.On("UpdateUser", mock.Anything, mockUser.Id, mock.Anything).Return(dbError).Once()

		result, err := env.AuthService.VerifyAccount(context.Background(), "test@example.com", "123456")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)
		assert.Nil(t, result)

		env.OTPService.AssertExpectations(t)
		env.UserRepository.AssertExpectations(t)
	})
}

func TestResendVerificationOTP(t *testing.T) {

	t.Run("Resend success", func(t *testing.T) {

		env := setupTest()

		mockUser := &domain.User{
			Email:  "unverified@example.com",
			Name:   "Test User",
			Status: "unverified",
		}

		env.UserRepository.On("GetUserByEmail", mock.Anything, "unverified@example.com").Return(mockUser, nil).Once()
		env.OTPService.On("Generate", mock.Anything, "unverified@example.com").Return("654321", nil).Once()
		env.EmailService.On("ResendOTP", mock.Anything, "unverified@example.com", "Test User", "654321").Return(nil).Once()

		err := env.AuthService.ResendVerificationOTP(context.Background(), "unverified@example.com")

		assert.NoError(t, err)

		env.UserRepository.AssertExpectations(t)
		env.OTPService.AssertExpectations(t)
		env.EmailService.AssertExpectations(t)
	})

	t.Run("Resend for non-existent user returns no error", func(t *testing.T) {

		env := setupTest()

		env.UserRepository.On("GetUserByEmail", mock.Anything, "notfound@example.com").Return(nil, user.ErrUserNotFound).Once()

		err := env.AuthService.ResendVerificationOTP(context.Background(), "notfound@example.com")

		assert.NoError(t, err)

		env.UserRepository.AssertExpectations(t)
		env.OTPService.AssertNotCalled(t, "Generate", mock.Anything, mock.Anything)
		env.EmailService.AssertNotCalled(t, "ResendOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Resend failed: Account is already verified", func(t *testing.T) {

		env := setupTest()

		mockUser := &domain.User{
			Email:  "verified@example.com",
			Status: "verified", // Tài khoản đã xác thực
		}
		env.UserRepository.On("GetUserByEmail", mock.Anything, "verified@example.com").Return(mockUser, nil).Once()

		err := env.AuthService.ResendVerificationOTP(context.Background(), "verified@example.com")

		assert.Error(t, err)
		assert.Equal(t, ErrAccountAlreadyVerified, err)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Resend failed: OTP generation fails", func(t *testing.T) {

		env := setupTest()

		mockUser := &domain.User{
			Email:  "unverified@example.com",
			Status: "unverified",
		}
		otpError := errors.New("redis is down")

		env.UserRepository.On("GetUserByEmail", mock.Anything, "unverified@example.com").Return(mockUser, nil).Once()
		env.OTPService.On("Generate", mock.Anything, "unverified@example.com").Return("", otpError).Once()

		err := env.AuthService.ResendVerificationOTP(context.Background(), "unverified@example.com")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)

		env.UserRepository.AssertExpectations(t)
		env.OTPService.AssertExpectations(t)
	})

	t.Run("Resend failed: Sending email fails", func(t *testing.T) {

		env := setupTest()

		mockUser := &domain.User{
			Email:  "unverified@example.com",
			Name:   "Test User",
			Status: "unverified",
		}
		mailError := errors.New("SMTP server is down")

		env.UserRepository.On("GetUserByEmail", mock.Anything, "unverified@example.com").Return(mockUser, nil).Once()
		env.OTPService.On("Generate", mock.Anything, "unverified@example.com").Return("654321", nil).Once()
		env.EmailService.On("ResendOTP", mock.Anything, "unverified@example.com", "Test User", "654321").Return(mailError).Once()

		err := env.AuthService.ResendVerificationOTP(context.Background(), "unverified@example.com")

		assert.Error(t, err)
		assert.Equal(t, common.ErrInternalServer, err)

		env.UserRepository.AssertExpectations(t)
		env.OTPService.AssertExpectations(t)
		env.EmailService.AssertExpectations(t)
	})
}

func TestSetPasswordForUser(t *testing.T) {

	t.Run("Set password success", func(t *testing.T) {

		env := setupTest()

		userID := uuid.New()
		newPassword := "my-new-strong-password"

		env.UserRepository.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(nil).Once()

		err := env.AuthService.SetPasswordForUser(context.Background(), userID, newPassword)

		assert.NoError(t, err)

		env.UserRepository.AssertExpectations(t)
	})

	t.Run("Set password failed: Repository error", func(t *testing.T) {

		env := setupTest()

		userID := uuid.New()
		dbError := errors.New("failed to update password in DB")

		env.UserRepository.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string")).Return(dbError).Once()

		err := env.AuthService.SetPasswordForUser(context.Background(), userID, "any-password")

		assert.Error(t, err)
		assert.Equal(t, dbError, err)

		env.UserRepository.AssertExpectations(t)
	})
}

func TestLoginWithGoogleIDToken(t *testing.T) {

	googleUser := domain.GoogleUser{
		Email:   "test.user@google.com",
		Name:    "Test Google User",
		Sub:     "google-id-123",
		Picture: "http://example.com/pic.jpg",
	}

	t.Run("Login success: Existing user", func(t *testing.T) {

		env := setupTest()
		existingUser := &domain.User{
			Id:       uuid.New(),
			Email:    googleUser.Email,
			Name:     "Old Name",
			Provider: "Google",
		}

		env.GoogleVerifier.On("VerifyGoogleIDToken", mock.Anything, "valid-google-token").Return(&googleUser, nil).Once()
		env.UserRepository.On("GetUserByEmail", mock.Anything, googleUser.Email).Return(existingUser, nil).Once()
		env.TokenService.On("GenerateAccessToken", existingUser).Return("access_token", nil).Once()
		env.TokenService.On("GenerateRefreshToken", mock.Anything, existingUser).Return("refresh_token", nil).Once()

		result, err := env.AuthService.LoginWithGoogleIDToken(context.Background(), "valid-google-token")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existingUser.Id, result.User.Id)
		assert.Equal(t, "access_token", result.AccessToken)

		env.GoogleVerifier.AssertExpectations(t)
		env.UserRepository.AssertExpectations(t)
		env.TokenService.AssertExpectations(t)
	})

	t.Run("Login success: New user registration", func(t *testing.T) {

		env := setupTest()

		env.GoogleVerifier.On("VerifyGoogleIDToken", mock.Anything, "valid-google-token").Return(&googleUser, nil).Once()
		env.UserRepository.On("GetUserByEmail", mock.Anything, googleUser.Email).Return(nil, user.ErrUserNotFound).Once()
		env.UserRepository.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		env.TokenService.On("GenerateAccessToken", mock.AnythingOfType("*domain.User")).Return("access_token", nil).Once()
		env.TokenService.On("GenerateRefreshToken", mock.Anything, mock.AnythingOfType("*domain.User")).Return("refresh_token", nil).Once()

		result, err := env.AuthService.LoginWithGoogleIDToken(context.Background(), "valid-google-token")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, googleUser.Email, result.User.Email)
		assert.Equal(t, "Google", result.User.Provider)

		env.GoogleVerifier.AssertExpectations(t)
		env.UserRepository.AssertExpectations(t)
		env.TokenService.AssertExpectations(t)
	})

	t.Run("Login failed: Invalid Google ID token", func(t *testing.T) {

		env := setupTest()

		verifyError := errors.New("invalid token signature")
		env.GoogleVerifier.On("VerifyGoogleIDToken", mock.Anything, "invalid-token").Return(nil, verifyError).Once()

		result, err := env.AuthService.LoginWithGoogleIDToken(context.Background(), "invalid-token")

		assert.Error(t, err)
		assert.Equal(t, verifyError, err)
		assert.Nil(t, result)

		env.GoogleVerifier.AssertExpectations(t)
	})

	t.Run("Login failed: Create new user fails", func(t *testing.T) {

		env := setupTest()

		dbError := errors.New("database connection failed")
		env.GoogleVerifier.On("VerifyGoogleIDToken", mock.Anything, "valid-google-token").Return(&googleUser, nil).Once()
		env.UserRepository.On("GetUserByEmail", mock.Anything, googleUser.Email).Return(nil, user.ErrUserNotFound).Once()
		env.UserRepository.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User")).Return(dbError).Once()

		result, err := env.AuthService.LoginWithGoogleIDToken(context.Background(), "valid-google-token")

		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		assert.Nil(t, result)

		env.GoogleVerifier.AssertExpectations(t)
		env.UserRepository.AssertExpectations(t)
	})
}
