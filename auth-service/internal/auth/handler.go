package auth

import (
	"net/http"
	"strings"

	"auth-service/internal/common"
	"auth-service/internal/domain"
	webHelpers "auth-service/internal/web/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service      Service
	tokenService TokenService
}

func NewHandler(s Service, tokenService TokenService) *Handler {
	return &Handler{service: s, tokenService: tokenService}
}

type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type LoginSuccessResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *domain.User `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type RegisterResponse struct {
	Message string       `json:"message"`
	User    *domain.User `json:"user"`
}

type EmailLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type SetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

type VerifyRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// GoogleLogin handles user login via Google ID token.
// @Summary      Google Login
// @Description  Authenticates a user using a Google ID token and returns access/refresh tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        token body GoogleLoginRequest true "Google ID Token"
// @Success      200   {object}  LoginSuccessResponse
// @Router       /auth/google [post]
func (handler *Handler) GoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	result, err := handler.service.LoginWithGoogleIDToken(c.Request.Context(), req.IDToken)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, LoginSuccessResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User,
	})
}

// RefreshToken generates a new access token from a refresh token.
// @Summary      Refresh Access Token
// @Description  Provides a new access token if the given refresh token is valid.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        token body      RefreshTokenRequest  true  "Refresh Token"
// @Success      200   {object}  RefreshTokenResponse
// @Router       /auth/refresh [post]
func (handler *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	newAccessToken, err := handler.service.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		AccessToken: newAccessToken,
	})
}

// LogOut invalidates a refresh token.
// @Summary      User Logout
// @Description  Logs the user out by invalidating the provided refresh token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        token body      LogoutRequest        true  "Refresh Token"
// @Success      200   {object}  webHelpers.OKResponse
// @Router       /auth/logout [post]
func (handler *Handler) LogOut(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	err := handler.service.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Logged out successfully"})
}

// Register creates a new user account.
// @Summary      Register New User
// @Description  Creates a new user account with email, password, and name.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        user  body      RegisterRequest      true  "User registration info"
// @Success      201  {object}  domain.User
// @Failure      404  {object}  webHelpers.ErrorResponse "user not found"
// @Failure      409  {object}  webHelpers.ErrorResponse "email is already registered"
// @Failure      500  {object}  webHelpers.ErrorResponse "an unexpected error occurred"
// @Router       /auth/register [post]
func (handler *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	createdUser, err := handler.service.RegisterWithEmail(c.Request.Context(), req.Email, req.Password, req.Name)

	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		Message: "Registration successful. Please check your email to verify your account.",
		User:    createdUser,
	})
}

// @Summary      Verify Account & Login
// @Description  Verifies OTP and logs the user in by returning tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        verification  body      VerifyRequest        true  "Email and OTP"
// @Success      200           {object}  LoginResult
// @Failure      400           {object}  webHelpers.ErrorResponse "Invalid OTP or request"
// @Failure      404           {object}  webHelpers.ErrorResponse "OTP not found or expired"
// @Router       /auth/otp/verify [post]
func (handler *Handler) VerifyAccount(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidBody)
		return
	}

	loginResult, err := handler.service.VerifyAccount(c.Request.Context(), req.Email, req.OTP)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, loginResult)
}

// @Summary      Resend OTP
// @Description  Generates and sends a new OTP to the user's email if the account is unverified.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        email_info body      ResendOTPRequest     true  "User's Email"
// @Success      200        {object}  webHelpers.OKResponse "Always returns a success message for security reasons"
// @Failure      400        {object}  webHelpers.ErrorResponse
// @Failure      409        {object}  webHelpers.ErrorResponse "Returned if account is already verified"
// @Router       /auth/otp/resend [post]
func (handler *Handler) ResendOTP(c *gin.Context) {
	var req ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidBody)
		return
	}

	err := handler.service.ResendVerificationOTP(c.Request.Context(), req.Email)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "If your account exists and hasn`t been verified yet, we`ve sent you a new OTP. Please check your email."})
}

// AppLogin handles user login via email and password.
// @Summary      Email/Password Login
// @Description  Authenticates a user using email and password, returns access/refresh tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        credentials body      EmailLoginRequest    true  "User Credentials"
// @Success      200         {object}  LoginSuccessResponse
// @Router       /auth/app [post]
func (handler *Handler) AppLogin(c *gin.Context) {
	var req EmailLoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	result, err := handler.service.LoginWithEmail(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, LoginSuccessResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         result.User,
	})
}

// @Summary      Set User Password
// @Description  Sets a password for the currently authenticated user. Requires Bearer token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        password body      SetPasswordRequest     true  "New Password"
// @Success      200      {object}  webHelpers.OKResponse
// @Router       /auth/password [post]
func (handler *Handler) SetPassword(c *gin.Context) {
	userIdfromToken, exists := c.Get(string(common.UserIdKey))
	if !exists {
		handler.handleError(c, common.ErrMissingContextParam)
		return
	}

	var req struct {
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	id, err := uuid.Parse(userIdfromToken.(string))
	if err != nil {
		handler.handleError(c, common.ErrInvalidUUIDFormat)
		return
	}

	if err := handler.service.SetPasswordForUser(c.Request.Context(), id, req.Password); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "password set successfully"})
}

func (handler *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			handler.handleError(c, &common.BusinessError{Code: 401, Message: "Missing Authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			handler.handleError(c, &common.BusinessError{Code: 401, Message: "Invalid Authorization header format"})
			c.Abort()
			return
		}

		claims, err := handler.tokenService.ValidateToken(parts[1])
		if err != nil {
			handler.handleError(c, &common.BusinessError{Code: 401, Message: "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set(string(common.EmailKey), claims["email"])
		c.Set(string(common.UserIdKey), claims["sub"])
		c.Set(string(common.Role), claims["role"])
		c.Next()
	}
}

// handleError is a helper function to return an error response to the client.
func (handler *Handler) handleError(c *gin.Context, err error) {
	webHelpers.HandleError(c, err)
}
