package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func ReadyHandler(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		err := rdb.Ping(ctx).Err()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"redis":  "error",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"redis":  "ok",
		})
	}
}
