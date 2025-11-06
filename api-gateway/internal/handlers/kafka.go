package handlers

import (
	"api-gateway/internal/kafka"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func KafkaSendHandler(producer kafka.Producer, topic string) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
			return
		}

		key := []byte(c.GetHeader("X-Message-Key"))

		if err := producer.Send(c.Request.Context(), topic, key, body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot send to kafka"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
