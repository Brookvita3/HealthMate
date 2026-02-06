package group

import (
	"auth-service/internal/member"
	"auth-service/internal/permission"
	h "auth-service/internal/web/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
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
// No group validation middleware needed - creating a new group.
func (handler *Handler) CreateGroup(c *gin.Context) {
	ownerID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description *string `json:"description"`
	}
	if !h.BindJSON(c, &req) {
		return
	}

	group, err := handler.groupService.CreateGroup(c.Request.Context(), ownerID, req.Name, req.Description)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondCreated(c, group)
}

// InviteMember handles inviting a user to a group.
// Uses pre-validated groupID from middleware.
func (handler *Handler) InviteMember(c *gin.Context) {
	inviterID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	// Use pre-validated groupID from middleware
	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		// Fallback to parsing if middleware wasn't applied
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if !h.BindJSON(c, &req) {
		return
	}

	targetUserID, ok := h.ParseBodyUUID(c, req.UserID)
	if !ok {
		return
	}

	if err := handler.memberService.InviteMember(c.Request.Context(), groupID, targetUserID, inviterID); err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondOK(c, "Invitation sent")
}

// AcceptInvitation handles a user accepting an invitation to join a group.
// Uses pre-validated groupID from middleware.
func (handler *Handler) AcceptInvitation(c *gin.Context) {
	myID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	if err := handler.memberService.AcceptInvitation(c.Request.Context(), groupID, myID); err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondOK(c, "Invitation accepted")
}

// RejectInvitation handles a user rejecting an invitation to join a group.
// Uses pre-validated groupID from middleware.
func (handler *Handler) RejectInvitation(c *gin.Context) {
	myID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	if err := handler.memberService.RejectInvitation(c.Request.Context(), groupID, myID); err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondOK(c, "Invitation rejected")
}

// RemoveMember handles removing a member from a group (Kick).
// Uses pre-validated groupID and memberID from middleware.
func (handler *Handler) RemoveMember(c *gin.Context) {
	requesterID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	// Use pre-validated memberID from middleware
	targetUserID, ok := h.GetValidatedMemberID(c)
	if !ok {
		targetUserID, ok = h.GetMemberID(c)
		if !ok {
			return
		}
	}

	if err := handler.memberService.RemoveMember(c.Request.Context(), groupID, targetUserID, requesterID); err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondOK(c, "Member removed")
}

// LeaveGroup handles the current user leaving a group.
// Uses pre-validated groupID from middleware.
func (handler *Handler) LeaveGroup(c *gin.Context) {
	myID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	if err := handler.memberService.LeaveGroup(c.Request.Context(), groupID, myID); err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondOK(c, "Left group successfully")
}

// GetPermissions handles retrieving shared permissions for the current user in a group.
// Uses pre-validated groupID from middleware.
func (handler *Handler) GetPermissions(c *gin.Context) {
	myID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	perms, err := handler.permService.GetPermissions(c.Request.Context(), groupID, myID)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondData(c, perms)
}

// GetMembers handles retrieving all members of a group.
// Uses pre-validated groupID from middleware.
func (handler *Handler) GetMembers(c *gin.Context) {
	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	members, err := handler.memberService.GetMembers(c.Request.Context(), groupID)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondData(c, members)
}

// SetPermission handles enabling or disabling a sharing permission for a member.
// Uses pre-validated groupID from middleware.
func (handler *Handler) SetPermission(c *gin.Context) {
	myID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	var req struct {
		MetricType string `json:"metric_type" binding:"required"`
		Enabled    bool   `json:"enabled"`
	}
	if !h.BindJSON(c, &req) {
		return
	}

	var err error
	if req.Enabled {
		err = handler.permService.EnableSharing(c.Request.Context(), groupID, myID, req.MetricType)
	} else {
		err = handler.permService.DisableSharing(c.Request.Context(), groupID, myID, req.MetricType)
	}

	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondOK(c, "Permission updated")
}

// UpdatePermissions handles updating all sharing permissions for the current user in a group.
// Uses pre-validated groupID from middleware.
func (handler *Handler) UpdatePermissions(c *gin.Context) {
	myID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	var req struct {
		MetricTypes []string `json:"metric_types" binding:"required"`
	}
	if !h.BindJSON(c, &req) {
		return
	}

	if err := handler.permService.UpdateSharing(c.Request.Context(), groupID, myID, req.MetricTypes); err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondOK(c, "Permissions updated")
}

// ListMyGroups returns all groups where the authenticated user is the owner.
// No group validation middleware needed - listing user's groups.
func (handler *Handler) ListMyGroups(c *gin.Context) {
	myID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groups, err := handler.groupService.ListUserGroups(c.Request.Context(), myID, 100, 0)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, groups)
}

// TransferOwnership handles transferring group ownership to another user.
// Uses pre-validated groupID from middleware.
func (handler *Handler) TransferOwnership(c *gin.Context) {
	currentOwnerID, ok := h.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := h.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = h.GetGroupID(c)
		if !ok {
			return
		}
	}

	var req struct {
		NewOwnerID string `json:"new_owner_id" binding:"required"`
	}
	if !h.BindJSON(c, &req) {
		return
	}

	newOwnerID, ok := h.ParseBodyUUID(c, req.NewOwnerID)
	if !ok {
		return
	}

	if err := handler.groupService.TransferOwnership(c.Request.Context(), groupID, currentOwnerID, newOwnerID); err != nil {
		h.HandleError(c, err)
		return
	}

	h.RespondOK(c, "Ownership transferred successfully")
}
