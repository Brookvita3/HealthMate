package group

import (
	"auth-service/internal/common"
	_ "auth-service/internal/domain"
	"auth-service/internal/member"
	"auth-service/internal/permission"
	webHelpers "auth-service/internal/web/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	groupService  Service
	memberService member.Service
	permService   permission.Service
}

type CreateGroupRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

type UpdateGroupRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type InviteMemberRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type UpdateMemberStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=accepted rejected"`
}

type SetPermissionRequest struct {
	MetricType string `json:"metric_type" binding:"required"`
	Enabled    bool   `json:"enabled"`
}

type UpdatePermissionsRequest struct {
	MetricTypes []string `json:"metric_types" binding:"required"`
}

type TransferOwnershipRequest struct {
	NewOwnerID string `json:"new_owner_id" binding:"required"`
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
// @Summary Create a new group
// @Description Create a new group with the current user as owner
// @Tags groups
// @Accept json
// @Produce json
// @Param request body CreateGroupRequest true "Group creation details"
// @Success 201 {object} domain.Group
// @Failure 400 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups [post]
func (handler *Handler) CreateGroup(c *gin.Context) {
	ownerID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	group, err := handler.groupService.CreateGroup(c.Request.Context(), ownerID, req.Name, req.Description)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, group)
}

// UpdateGroup handles updating group information.
// @Summary Update group info
// @Description Update name and description of a group
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param request body UpdateGroupRequest true "Update details"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 400,403,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id} [put]
func (handler *Handler) UpdateGroup(c *gin.Context) {
	requesterID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	if err := handler.groupService.UpdateGroup(c.Request.Context(), groupID, req.Name, req.Description, requesterID); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Group updated successfully"})
}

// DeleteGroup handles deleting a group.
// @Summary Delete a group
// @Description Remove a group (Owner only)
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 403,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id} [delete]
func (handler *Handler) DeleteGroup(c *gin.Context) {
	requesterID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		return
	}

	if err := handler.groupService.DeleteGroup(c.Request.Context(), groupID, requesterID); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Group deleted successfully"})
}

// InviteMember handles inviting a user to a group.
// @Summary Invite a member
// @Description Send an invitation to a user by email
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param request body InviteMemberRequest true "Invite details"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 400,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/members [post]
func (handler *Handler) InviteMember(c *gin.Context) {
	inviterID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	// Use pre-validated groupID from middleware
	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		// Fallback to parsing if middleware wasn't applied
		groupID, ok = webHelpers.GetGroupID(c)
		if !ok {
			return
		}
	}

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	if err := handler.memberService.InviteByEmail(c.Request.Context(), groupID, req.Email, inviterID); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Invitation sent"})
}

// UpdateMyMemberStatus handles updating the current user's membership status (Accept/Reject).
// @Summary Update membership status
// @Description Accept or reject a group invitation
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param request body UpdateMemberStatusRequest true "Status update"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 400,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/members/me [put]
func (handler *Handler) UpdateMyMemberStatus(c *gin.Context) {
	myID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		return
	}

	var req UpdateMemberStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	if err := handler.memberService.UpdateMemberStatus(c.Request.Context(), groupID, myID, req.Status); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Membership status updated"})
}

// RemoveMember handles removing a member from a group (Kick).
// @Summary Remove a member
// @Description Kick a member from the group (Owner only)
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Param member_id path string true "Member User ID"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 403,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/members/{member_id} [delete]
func (handler *Handler) RemoveMember(c *gin.Context) {
	requesterID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = webHelpers.GetGroupID(c)
		if !ok {
			return
		}
	}

	// Use pre-validated memberID from middleware
	targetUserID, ok := webHelpers.GetValidatedMemberID(c)
	if !ok {
		targetUserID, ok = webHelpers.GetMemberID(c)
		if !ok {
			return
		}
	}

	if err := handler.memberService.RemoveMember(c.Request.Context(), groupID, targetUserID, requesterID); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Member removed"})
}

// LeaveGroup handles the current user leaving a group.
// @Summary Leave a group
// @Description Leave a group you are a member of
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 400,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/members/me [delete]
func (handler *Handler) LeaveGroup(c *gin.Context) {
	myID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = webHelpers.GetGroupID(c)
		if !ok {
			return
		}
	}

	if err := handler.memberService.LeaveGroup(c.Request.Context(), groupID, myID); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Left group successfully"})
}

// GetPermissions handles retrieving shared permissions for the current user in a group.
// @Summary Get my permissions
// @Description Get current user's sharing permissions in a group
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {array} domain.Permission
// @Failure 404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/permissions [get]
func (handler *Handler) GetPermissions(c *gin.Context) {
	myID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = webHelpers.GetGroupID(c)
		if !ok {
			return
		}
	}

	perms, err := handler.permService.GetPermissions(c.Request.Context(), groupID, myID)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, perms)
}

// GetMembers handles retrieving all members of a group.
// @Summary Get group members
// @Description List all members of a group
// @Tags groups
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {array} domain.GroupMember
// @Failure 404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/members [get]
func (handler *Handler) GetMembers(c *gin.Context) {
	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = webHelpers.GetGroupID(c)
		if !ok {
			return
		}
	}

	members, err := handler.memberService.GetMembers(c.Request.Context(), groupID)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, members)
}

// SetPermission handles enabling or disabling a sharing permission for a member.
// @Summary Set sharing permission
// @Description Enable or disable sharing for a specific metric type
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param request body SetPermissionRequest true "Permission details"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 400,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/permissions [post]
func (handler *Handler) SetPermission(c *gin.Context) {
	myID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = webHelpers.GetGroupID(c)
		if !ok {
			return
		}
	}

	var req SetPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	var err error
	if req.Enabled {
		err = handler.permService.EnableSharing(c.Request.Context(), groupID, myID, req.MetricType)
	} else {
		err = handler.permService.DisableSharing(c.Request.Context(), groupID, myID, req.MetricType)
	}

	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Permission updated"})
}

// UpdatePermissions handles updating all sharing permissions for the current user in a group.
// @Summary Update all permissions
// @Description Update all sharing permissions for the current user
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param request body UpdatePermissionsRequest true "List of metric types"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 400,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/permissions [put]
func (handler *Handler) UpdatePermissions(c *gin.Context) {
	myID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = webHelpers.GetGroupID(c)
		if !ok {
			return
		}
	}

	var req UpdatePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	if err := handler.permService.UpdateSharing(c.Request.Context(), groupID, myID, req.MetricTypes); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Permissions updated"})
}

// ListMyGroups returns all groups the user belongs to.
// @Summary List my groups
// @Description Get all groups (owned or joined) for the current user
// @Tags groups
// @Produce json
// @Success 200 {array} domain.Group
// @Failure 401 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups [get]
func (handler *Handler) ListMyGroups(c *gin.Context) {
	myID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groups, err := handler.groupService.ListUserGroups(c.Request.Context(), myID, 100, 0)
	if err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, groups)
}

// TransferOwnership handles transferring group ownership to another user.
// @Summary Transfer ownership
// @Description Transfer group ownership to a new user
// @Tags groups
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param request body TransferOwnershipRequest true "New owner details"
// @Success 200 {object} webHelpers.OKResponse
// @Failure 400,403,404 {object} webHelpers.ErrorResponse
// @Security BearerAuth
// @Router /groups/{id}/owner [put]
func (handler *Handler) TransferOwnership(c *gin.Context) {
	currentOwnerID, ok := webHelpers.GetAuthUserID(c)
	if !ok {
		return
	}

	groupID, ok := webHelpers.GetValidatedGroupID(c)
	if !ok {
		groupID, ok = webHelpers.GetGroupID(c)
		if !ok {
			return
		}
	}

	var req TransferOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.handleError(c, common.ErrInvalidRequest)
		return
	}

	newOwnerID, err := uuid.Parse(req.NewOwnerID)
	if err != nil {
		handler.handleError(c, common.ErrInvalidUUIDFormat)
		return
	}

	if err := handler.groupService.TransferOwnership(c.Request.Context(), groupID, currentOwnerID, newOwnerID); err != nil {
		handler.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "Ownership transferred successfully"})
}

// handleError is a helper function to return an error response to the client.
func (handler *Handler) handleError(c *gin.Context, err error) {
	webHelpers.HandleError(c, err)
}
