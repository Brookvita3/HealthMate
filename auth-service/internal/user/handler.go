package user

import (
	"net/http"
	"strconv"

	"auth-service/internal/common"
	"auth-service/internal/domain"

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
	userId, exists := c.Get(string(common.UserIdKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": common.ErrMissingContextParam.Error()})
		return
	}

	id, err := uuid.Parse(userId.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	profile, err := h.service.GetUserProfile(c.Request.Context(), id)
	if err != nil {
		if err == ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": common.ErrInternalServer.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userId, exists := c.Get(string(common.UserIdKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": common.ErrMissingContextParam.Error()})
		return
	}
	id, err := uuid.Parse(userId.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	var req UpdateUserParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	if err := h.service.UpdateUserProfile(c.Request.Context(), id, req); err != nil {
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
		Status: "active",
	}

	users, err := h.service.ListUsers(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if users == nil {
		users = []domain.User{}
	}

	c.JSON(http.StatusOK, users)
}
