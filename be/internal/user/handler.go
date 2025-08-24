package user

import (
	"net/http"
	"strconv"

	"healthmate/internal/common" // Assuming this is where your context keys are defined

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultLimit  = 20
	defaultOffset = 0
)

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) GetProfile(c *gin.Context) {
	userIDStr, exists := c.Get(string(common.UserIdKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in token context"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format in token"})
		return
	}

	profile, err := h.service.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userIDStr, _ := c.Get(string(common.UserIdKey))
	userID, _ := uuid.Parse(userIDStr.(string))

	var req UpdateUserParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if err := h.service.UpdateUserProfile(c.Request.Context(), userID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) ListUsers(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if err != nil || limit < 0 {
		limit = defaultLimit
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", strconv.Itoa(defaultOffset)))
	if err != nil || offset < 0 {
		offset = defaultOffset
	}

	search := c.Query("search")

	params := ListUsersParams{
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	users, err := h.service.ListUsers(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	if users == nil {
		users = []User{}
	}

	c.JSON(http.StatusOK, users)
}
