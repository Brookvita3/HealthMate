package helpers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetValidatedGroupID retrieves the pre-validated groupID from context.
// This should be used after ValidateGroupExists middleware has run.
func GetValidatedGroupID(c *gin.Context) (uuid.UUID, bool) {
	groupIDRaw, exists := c.Get("validated_group_id")
	if !exists {
		return uuid.Nil, false
	}
	return groupIDRaw.(uuid.UUID), true
}

// GetValidatedMemberID retrieves the pre-validated memberID from context.
// This should be used after ValidateMemberExists middleware has run.
func GetValidatedMemberID(c *gin.Context) (uuid.UUID, bool) {
	memberIDRaw, exists := c.Get("validated_member_id")
	if !exists {
		return uuid.Nil, false
	}
	return memberIDRaw.(uuid.UUID), true
}
