package data

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
}

func NewDataHandler() *Handler {
	return &Handler{}
}

func (h *Handler) SendData(c *gin.Context) {
	type HealthData struct {
		Type      string `json:"type"`
		Value     int64  `json:"value"`
		Timestamp int64  `json:"timestamp"`
	}

	var req []HealthData
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	log.Println(req)

	c.Status(http.StatusNoContent)
}
