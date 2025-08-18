package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DataHandler struct {
}

func NewDataHandler() *DataHandler {
	return &DataHandler{}
}

func (h *DataHandler) SendData(c *gin.Context) {
	var req struct {
		Beats int `json:"beats"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	log.Println(req.Beats)

	c.JSON(http.StatusOK, gin.H{
		"steps": req.Beats,
	})
}
