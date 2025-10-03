package auth

import (
	"errors"
	"net/http"
	"strings"

	"healthmate/internal/common"
	"healthmate/internal/user"

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

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessMessageResponse struct {
	Message string `json:"message"`
}

type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type LoginSuccessResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	User         *user.User `json:"user"`
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
	Message string     `json:"message"`
	User    *user.User `json:"user"`
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
func (h *Handler) GoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidRequest.Error()})
		return
	}

	result, err := h.service.LoginWithGoogleIDToken(c.Request.Context(), req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
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
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidRequest.Error()})
		return
	}

	newAccessToken, err := h.service.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
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
// @Success      200   {object}  SuccessMessageResponse
// @Router       /auth/logout [post]
func (h *Handler) LogOut(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidRequest.Error()})
		return
	}

	err := h.service.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessMessageResponse{Message: "Logged out successfully"})
}

// Register creates a new user account.
// @Summary      Register New User
// @Description  Creates a new user account with email, password, and name.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        user  body      RegisterRequest      true  "User registration info"
// @Success      201  {object}  user.User
// @Failure      404  {object}  ErrorResponse "user not found"
// @Failure      409  {object}  ErrorResponse "email is already registered"
// @Failure      500  {object}  ErrorResponse "an unexpected error occurred"
// @Router       /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidRequest.Error()})
		return
	}

	createdUser, err := h.service.RegisterWithEmail(c.Request.Context(), req.Email, req.Password, req.Name)

	if err != nil {
		h.handleError(c, err)
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
// @Failure      400           {object}  ErrorResponse "Invalid OTP or request"
// @Failure      404           {object}  ErrorResponse "OTP not found or expired"
// @Router       /auth/otp/verify [post]
func (h *Handler) VerifyAccount(c *gin.Context) {
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidBody.Error()})
		return
	}

	loginResult, err := h.service.VerifyAccount(c.Request.Context(), req.Email, req.OTP)
	if err != nil {
		h.handleError(c, err)
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
// @Success      200        {object}  SuccessMessageResponse "Always returns a success message for security reasons"
// @Failure      400        {object}  ErrorResponse
// @Failure      409        {object}  ErrorResponse "Returned if account is already verified"
// @Router       /auth/otp/resend [post]
func (h *Handler) ResendOTP(c *gin.Context) {
	var req ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidBody.Error()})
		return
	}

	err := h.service.ResendVerificationOTP(c.Request.Context(), req.Email)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessMessageResponse{Message: "If your account exists and hasn`t been verified yet, we`ve sent you a new OTP. Please check your email."})
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
func (h *Handler) AppLogin(c *gin.Context) {
	var req EmailLoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidRequest.Error()})
		return
	}

	result, err := h.service.LoginWithEmail(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
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
// @Success      200      {object}  SuccessMessageResponse
// @Router       /auth/password [post]
func (h *Handler) SetPassword(c *gin.Context) {
	userIdfromToken, exists := c.Get(string(common.UserIdKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: common.ErrMissingContextParam.Error()})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidRequest.Error()})
		return
	}

	id, err := uuid.Parse(userIdfromToken.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: common.ErrInvalidUUIDFormat.Error()})
		return
	}

	if err := h.service.SetPasswordForUser(c.Request.Context(), id, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessMessageResponse{Message: "password set successfully"})
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Missing Authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid Authorization header format"})
			c.Abort()
			return
		}

		claims, err := h.tokenService.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid or expired token"})
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
// It checks if the error is a BusinessError and if so, returns a JSON response with the error code and message.
// If the error is not a BusinessError, it returns a JSON response with a generic error message and a 500 status code.
func (h *Handler) handleError(c *gin.Context, err error) {

	var businessErr *common.BusinessError

	if errors.As(err, &businessErr) {
		c.JSON(businessErr.Code, ErrorResponse{Error: businessErr.Message})
	} else {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "an unexpected error occurred"})
	}
}
