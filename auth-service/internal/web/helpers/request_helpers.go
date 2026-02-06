package helpers

import (
	"auth-service/internal/common"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetAuthUserID extracts and parses the authenticated user's ID from context.
// Returns the UUID and true if successful, otherwise responds with error and returns false.
func GetAuthUserID(c *gin.Context) (uuid.UUID, bool) {
	userIdRaw, exists := c.Get(string(common.UserIdKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(userIdRaw.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return uuid.Nil, false
	}

	return userID, true
}

// GetParamUUID extracts and parses a UUID from URL parameters.
// Returns the UUID and true if successful, otherwise responds with error and returns false.
func GetParamUUID(c *gin.Context, paramName string) (uuid.UUID, bool) {
	paramValue := c.Param(paramName)
	parsedID, err := uuid.Parse(paramValue)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return uuid.Nil, false
	}
	return parsedID, true
}

// GetGroupID is a convenience wrapper for getting the "id" param as group UUID.
func GetGroupID(c *gin.Context) (uuid.UUID, bool) {
	return GetParamUUID(c, "id")
}

// GetMemberID is a convenience wrapper for getting the "member_id" param as member UUID.
func GetMemberID(c *gin.Context) (uuid.UUID, bool) {
	return GetParamUUID(c, "member_id")
}

// ParseBodyUUID parses a UUID string from request body field.
// Returns the UUID and true if successful, otherwise responds with error and returns false.
func ParseBodyUUID(c *gin.Context, uuidStr string) (uuid.UUID, bool) {
	parsedID, err := uuid.Parse(uuidStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return uuid.Nil, false
	}
	return parsedID, true
}

// BindJSON binds JSON body to the provided struct.
// Returns true if successful, otherwise responds with error and returns false.
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidRequest.Error()})
		return false
	}
	return true
}
