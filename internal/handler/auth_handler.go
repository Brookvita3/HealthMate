package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heathhub/internal/service"
)

type AuthHandler struct {
	AuthService *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{AuthService: s}
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	result, err := h.AuthService.LoginWithGoogleIDToken(req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":       result.AccessToken,
		"expires_in":         result.AccessTTL,
		"refresh_token":      result.RefreshToken,
		"refresh_expires_in": result.RefreshTTL,
		"user":               result.User,
	})
}
