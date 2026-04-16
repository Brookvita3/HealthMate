package medication

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// getUserID extracts user_id from JWT claim "sub" set by middleware.
func getUserID(c *gin.Context) (string, bool) {
	userID := c.GetString("sub")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in token"})
		return "", false
	}
	return userID, true
}

// ListMedications handles GET /medications
// @Summary List all medications for the user
// @Description Get a list of medications with nested reminders for the authenticated user
// @Tags medications
// @Accept json
// @Produce json
// @Success 200 {array} Medication
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /medications [get]
// @Security BearerAuth
func (h *Handler) ListMedications(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	meds, err := h.service.ListMedications(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to list medications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list medications"})
		return
	}

	c.JSON(http.StatusOK, meds)
}

// CreateMedication handles POST /medications
// @Summary Create a new medication
// @Description Add a new medication and its associated reminders for the authenticated user
// @Tags medications
// @Accept json
// @Produce json
// @Param medication body CreateMedicationRequest true "Medication creation request"
// @Success 201 {object} Medication
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /medications [post]
// @Security BearerAuth
func (h *Handler) CreateMedication(c *gin.Context) {
	var req CreateMedicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	med, err := h.service.CreateMedication(c.Request.Context(), userID, req)
	if err != nil {
		log.Printf("Failed to create medication: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create medication"})
		return
	}

	c.JSON(http.StatusCreated, med)
}

// DeleteMedication handles DELETE /medications/:medicationId
// @Summary Delete a medication
// @Description Delete a specific medication and all its associated reminders by ID
// @Tags medications
// @Accept json
// @Produce json
// @Param medicationId path string true "Medication ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /medications/{medicationId} [delete]
// @Security BearerAuth
func (h *Handler) DeleteMedication(c *gin.Context) {
	medicationID := c.Param("medicationId")

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	err := h.service.DeleteMedication(c.Request.Context(), medicationID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "medication not found"})
			return
		}
		log.Printf("Failed to delete medication: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete medication"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "medication deleted"})
}

// TakeMedication handles POST /medications/:medicationId/reminders/:reminderId/take
// @Summary Record medication taken
// @Description Toggle the 'last_taken' status of a specific reminder and return the updated medication list
// @Tags medications
// @Accept json
// @Produce json
// @Param medicationId path string true "Medication ID"
// @Param reminderId path string true "Reminder ID"
// @Success 200 {array} Medication
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /medications/{medicationId}/reminders/{reminderId}/take [post]
// @Security BearerAuth
func (h *Handler) TakeMedication(c *gin.Context) {
	medicationID := c.Param("medicationId")
	reminderID := c.Param("reminderId")

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	meds, err := h.service.TakeMedication(c.Request.Context(), medicationID, reminderID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "medication or reminder not found"})
			return
		}
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "medication does not belong to this user"})
			return
		}
		log.Printf("Failed to take medication: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle take"})
		return
	}

	c.JSON(http.StatusOK, meds)
}

type RegisterDeviceTokenRequest struct {
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform"`
}

// RegisterDeviceToken handles POST /medications/device-token
// @Summary Register a device token
// @Description Save or update the device token for push notifications
// @Tags medications
// @Accept json
// @Produce json
// @Param token body RegisterDeviceTokenRequest true "Device token registration"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /medications/device-token [post]
// @Security BearerAuth
func (h *Handler) RegisterDeviceToken(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req RegisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.RegisterDeviceToken(c.Request.Context(), userID, req.Token, req.Platform)
	if err != nil {
		log.Printf("Failed to register device token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register device token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device token registered"})
}
