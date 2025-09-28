package auth

import (
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

type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// LoginSuccessResponse chứa token và thông tin người dùng khi đăng nhập thành công.
type LoginSuccessResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	User         user.User `json:"user"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	result, err := h.service.LoginWithGoogleIDToken(c.Request.Context(), req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginSuccessResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         *result.User,
	})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	newAccessToken, err := h.service.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": newAccessToken,
	})
}

func (h *Handler) LogOut(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	err := h.service.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Name     string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	user, err := h.service.RegisterWithEmail(c.Request.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) AppLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	result, err := h.service.LoginWithEmail(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"user":          result.User,
	})
}

func (h *Handler) SetPassword(c *gin.Context) {
	userIdfromToken, exists := c.Get(string(common.UserIdKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": common.ErrMissingContextParam.Error()})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	id, err := uuid.Parse(userIdfromToken.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	if err := h.service.SetPasswordForUser(c.Request.Context(), id, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password set successfully"})
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}

		claims, err := h.tokenService.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set(string(common.EmailKey), claims["email"])
		c.Set(string(common.UserIdKey), claims["sub"])
		c.Set(string(common.Role), claims["role"])
		c.Next()
	}
}
