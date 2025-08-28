package connection

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"healthmate/internal/common"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type CreateRequestPayload struct {
	UserId string `json:"userId" binding:"required,uuid"`
}

type RespondRequestPayload struct {
	Action string `json:"action" binding:"required,oneof=accept reject"`
}

func (h *Handler) SendRequest(c *gin.Context) {
	requesterIdStr, _ := c.Get(string(common.UserIdKey))
	requesterId, err := uuid.Parse(requesterIdStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	var payload CreateRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidBody.Error() + ": " + err.Error()})
		return
	}
	receiverId, _ := uuid.Parse(payload.UserId)

	err = h.service.CreateRequest(c.Request.Context(), requesterId, receiverId)
	if err != nil {
		if errors.Is(err, ErrConnectionExists) || errors.Is(err, ErrInvalidRequest) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": common.ErrInternalServer.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Connection request sent successfully"})
}

func (h *Handler) RespondToRequest(c *gin.Context) {
	acceptorIdStr, _ := c.Get(string(common.UserIdKey))
	acceptorId, err := uuid.Parse(acceptorIdStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	requesterIdStr := c.Param("requesterId")
	requesterId, err := uuid.Parse(requesterIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID format"})
		return
	}

	var payload RespondRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidBody.Error() + ": " + err.Error()})
		return
	}

	if payload.Action == "accept" {
		err = h.service.UpdateConnectionStatus(c.Request.Context(), acceptorId, requesterId, StatusAccepted)
	} else {
		err = h.service.DeleteConnection(c.Request.Context(), acceptorId, requesterId)
	}

	if err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if errors.Is(err, ErrInvalidStatus) || errors.Is(err, ErrInvalidRequest) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": common.ErrInternalServer.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request " + payload.Action + "ed successfully"})
}

func (h *Handler) GetConnectionByPair(c *gin.Context) {
	currentUserIdStr, _ := c.Get(string(common.UserIdKey))
	currentUserId, err := uuid.Parse(currentUserIdStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	targetUserIdStr := c.Param("userId")
	targetUserId, err := uuid.Parse(targetUserIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	conn, err := h.service.GetConnectionByPair(c.Request.Context(), currentUserId, targetUserId)
	if err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": common.ErrInternalServer.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, conn)
}

func (h *Handler) DeleteConnection(c *gin.Context) {
	currentUserIdStr, _ := c.Get(string(common.UserIdKey))
	currentUserId, err := uuid.Parse(currentUserIdStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": common.ErrInvalidUUIDFormat.Error()})
		return
	}

	targetUserIdStr := c.Param("userId")
	targetUserId, err := uuid.Parse(targetUserIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	err = h.service.DeleteConnection(c.Request.Context(), currentUserId, targetUserId)
	if err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": common.ErrInternalServer.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
