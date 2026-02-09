package user

import (
	"auth-service/internal/common"
	"auth-service/internal/domain"
	webHelpers "auth-service/internal/web/helpers"
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

// GetProfile returns the current user's profile.
// @Summary Get user profile
// @Description Get profile details of the currently authenticated user
// @Tags users
// @Produce json
// @Success 200 {object} domain.User
// @Failure 401,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /users/profile [get]
func (handler *Handler) GetProfile(c *gin.Context) {
	userId, exists := c.Get(string(common.UserIdKey))
	if !exists {
		handler.handleError(c, common.ErrMissingContextParam)
		return
	}

	id, err := uuid.Parse(userId.(string))
	if err != nil {
		handler.handleError(c, common.ErrInvalidUUIDFormat)
		return
	}

	profile, err := handler.service.GetUserProfile(c.Request.Context(), id)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile updates the current user's profile.
// @Summary Update user profile
// @Description Update profile details (name, picture) of the current user
// @Tags users
// @Accept json
// @Produce json
// @Param request body UpdateUserParams true "Profile updates"
// @Success 200 {string} string "OK"
// @Failure 400,401 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /users/profile [put]
func (handler *Handler) UpdateProfile(c *gin.Context) {
	userId, exists := c.Get(string(common.UserIdKey))
	if !exists {
		handler.handleError(c, common.ErrMissingContextParam)
		return
	}
	id, err := uuid.Parse(userId.(string))
	if err != nil {
		handler.handleError(c, common.ErrInvalidUUIDFormat)
		return
	}

	var req UpdateUserParams
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	if err := handler.service.UpdateUserProfile(c.Request.Context(), id, req); err != nil {
		handler.handleError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

// ListUsers searches for users by name or email.
// @Summary List/Search users
// @Description Retrieve a list of users with optional search and pagination
// @Tags users
// @Produce json
// @Param search query string false "Search query"
// @Param limit query int false "Limit (default 20)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {array} domain.User
// @Failure 500 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /users [get]
func (handler *Handler) ListUsers(c *gin.Context) {
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

	users, err := handler.service.ListUsers(c.Request.Context(), params)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	if users == nil {
		users = []domain.User{}
	}

	c.JSON(http.StatusOK, users)
}

// handleError is a helper function to return an error response to the client.
func (handler *Handler) handleError(c *gin.Context, err error) {
	webHelpers.HandleError(c, err)
}
