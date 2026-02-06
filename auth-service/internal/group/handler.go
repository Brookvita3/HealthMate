package group

import (
	"auth-service/internal/common"
	"auth-service/internal/member"
	"auth-service/internal/permission"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	groupService  Service
	memberService member.Service
	permService   permission.Service
}

// NewHandler creates a new instance of group.Handler with its dependencies.
func NewHandler(gs Service, ms member.Service, ps permission.Service) *Handler {
	return &Handler{
		groupService:  gs,
		memberService: ms,
		permService:   ps,
	}
}

// CreateGroup handles the creation of a new group.
// It extracts the owner ID from the authenticated context.
func (h *Handler) CreateGroup(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	ownerID := uuid.MustParse(userId.(string))

	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	group, err := h.groupService.CreateGroup(c.Request.Context(), ownerID, req.Name, req.Description)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, group)
}

// InviteMember handles inviting a user to a group.
// Requires group owner privileges.
func (h *Handler) InviteMember(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	inviterID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	err = h.memberService.InviteMember(c.Request.Context(), groupID, targetUserID, inviterID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation sent"})
}

// AcceptInvitation handles a user accepting an invitation to join a group.
func (h *Handler) AcceptInvitation(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	myID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	err = h.memberService.AcceptInvitation(c.Request.Context(), groupID, myID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation accepted"})
}

// RejectInvitation handles a user rejecting an invitation to join a group.
func (h *Handler) RejectInvitation(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	myID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	err = h.memberService.RejectInvitation(c.Request.Context(), groupID, myID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation rejected"})
}

// RemoveMember handles removing a member from a group (Kick).
func (h *Handler) RemoveMember(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	requesterID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	targetUserID, err := uuid.Parse(c.Param("member_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	err = h.memberService.RemoveMember(c.Request.Context(), groupID, targetUserID, requesterID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// LeaveGroup handles the current user leaving a group.
func (h *Handler) LeaveGroup(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	myID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	err = h.memberService.LeaveGroup(c.Request.Context(), groupID, myID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left group successfully"})
}

// GetPermissions handles retrieving shared permissions for the current user in a group.
func (h *Handler) GetPermissions(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	myID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	perms, err := h.permService.GetPermissions(c.Request.Context(), groupID, myID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, perms)
}

// GetMembers handles retrieving all members of a group.
func (h *Handler) GetMembers(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	members, err := h.memberService.GetMembers(c.Request.Context(), groupID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, members)
}

// SetPermission handles enabling or disabling a sharing permission for a member.
func (h *Handler) SetPermission(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	myID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	var req struct {
		MetricType string `json:"metric_type" binding:"required"`
		Enabled    bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	if req.Enabled {
		err = h.permService.EnableSharing(c.Request.Context(), groupID, myID, req.MetricType)
	} else {
		err = h.permService.DisableSharing(c.Request.Context(), groupID, myID, req.MetricType)
	}

	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission updated"})
}

// UpdatePermissions handles updating all sharing permissions for the current user in a group.
func (h *Handler) UpdatePermissions(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	myID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	var req struct {
		MetricTypes []string `json:"metric_types" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	err = h.permService.UpdateSharing(c.Request.Context(), groupID, myID, req.MetricTypes)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions updated"})
}

// ListMyGroups returns all groups where the authenticated user is the owner.
func (h *Handler) ListMyGroups(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	myID := uuid.MustParse(userId.(string))

	groups, err := h.groupService.ListUserGroups(c.Request.Context(), myID, 100, 0)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, groups)
}

// TransferOwnership handles transferring group ownership to another user.
func (h *Handler) TransferOwnership(c *gin.Context) {
	userId, _ := c.Get(string(common.UserIdKey))
	currentOwnerID := uuid.MustParse(userId.(string))
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	var req struct {
		NewOwnerID string `json:"new_owner_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return
	}

	newOwnerID, err := uuid.Parse(req.NewOwnerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	err = h.groupService.TransferOwnership(c.Request.Context(), groupID, currentOwnerID, newOwnerID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ownership transferred successfully"})
}

// handleError is a helper to return consistent error responses based on BusinessError.
func (h *Handler) handleError(c *gin.Context, err error) {
	var businessErr *common.BusinessError
	if errors.As(err, &businessErr) {
		c.JSON(businessErr.Code, gin.H{"error": businessErr.Message})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "an unexpected error occurred"})
	}
}
