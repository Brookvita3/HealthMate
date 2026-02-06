package middleware

import (
	"auth-service/internal/common"
	"auth-service/internal/validation"
	h "auth-service/internal/web/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GroupMiddleware provides middleware for group-related validations
type GroupMiddleware struct {
	validator *validation.Validator
}

// NewGroupMiddleware creates a new GroupMiddleware instance
func NewGroupMiddleware(validator *validation.Validator) *GroupMiddleware {
	return &GroupMiddleware{validator: validator}
}

// ValidateGroupExists middleware checks if the group exists before proceeding
// It extracts groupID from the ":id" URL parameter
// If valid, it stores the parsed groupID in context as "validated_group_id"
func (m *GroupMiddleware) ValidateGroupExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, ok := h.GetGroupID(c)
		if !ok {
			c.Abort()
			return
		}

		if err := m.validator.ValidateGroupExists(c.Request.Context(), groupID); err != nil {
			h.HandleError(c, err)
			c.Abort()
			return
		}

		// Store validated groupID in context for handlers to use
		c.Set("validated_group_id", groupID)
		c.Next()
	}
}

// ValidateMemberExists middleware checks if the member exists in the group
// It extracts member_id from the ":member_id" URL parameter
// Requires ValidateGroupExists to be called first
func (m *GroupMiddleware) ValidateMemberExists() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get validated groupID from context (set by ValidateGroupExists)
		groupIDRaw, exists := c.Get("validated_group_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "group validation required"})
			c.Abort()
			return
		}
		groupID := groupIDRaw.(uuid.UUID)

		memberID, ok := h.GetMemberID(c)
		if !ok {
			c.Abort()
			return
		}

		if err := m.validator.ValidateMemberExists(c.Request.Context(), groupID, memberID); err != nil {
			h.HandleError(c, err)
			c.Abort()
			return
		}

		// Store validated memberID in context
		c.Set("validated_member_id", memberID)
		c.Next()
	}
}

// RequireGroupOwnership middleware checks if the authenticated user is the group owner
// Requires ValidateGroupExists to be called first and user to be authenticated
func (m *GroupMiddleware) RequireGroupOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get validated groupID from context
		groupIDRaw, exists := c.Get("validated_group_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "group validation required"})
			c.Abort()
			return
		}
		groupID := groupIDRaw.(uuid.UUID)

		// Get authenticated user ID from context
		userIDRaw, exists := c.Get(string(common.UserIdKey))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		userID, err := uuid.Parse(userIDRaw.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
			c.Abort()
			return
		}

		if err := m.validator.ValidateIsGroupOwner(c.Request.Context(), groupID, userID); err != nil {
			h.HandleError(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireGroupMembership middleware checks if the authenticated user is a member of the group
// Requires ValidateGroupExists to be called first and user to be authenticated
func (m *GroupMiddleware) RequireGroupMembership() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get validated groupID from context
		groupIDRaw, exists := c.Get("validated_group_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "group validation required"})
			c.Abort()
			return
		}
		groupID := groupIDRaw.(uuid.UUID)

		// Get authenticated user ID from context
		userIDRaw, exists := c.Get(string(common.UserIdKey))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		userID, err := uuid.Parse(userIDRaw.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
			c.Abort()
			return
		}

		if err := m.validator.ValidateMemberExists(c.Request.Context(), groupID, userID); err != nil {
			h.HandleError(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}
