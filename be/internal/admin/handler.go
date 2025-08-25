// Package admin provides HTTP handlers for administrative endpoints.
package admin

import (
	"healthmate/internal/user"
	"net/http"
	"strconv"

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
	status := c.Query("status")

	params := user.ListUsersParams{
		Search: search,
		Status: status,
		Limit:  limit,
		Offset: offset,
	}

	users, err := h.service.ListUsers(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	if users == nil {
		users = []user.User{}
	}

	c.JSON(http.StatusOK, users)
}

func (h *Handler) BanUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	if err := h.service.BanUser(c.Request.Context(), userID); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ban user"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) UnbanUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	if err := h.service.UnbanUser(c.Request.Context(), userID); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unban user"})
		return
	}

	c.Status(http.StatusNoContent)
}
