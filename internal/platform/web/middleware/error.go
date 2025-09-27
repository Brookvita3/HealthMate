package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %v", err)
				if !c.Writer.Written() {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Internal Server Error",
					})
				}
				c.Abort()
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Printf("[ERROR] %v", e.Err)
			}

			if !c.Writer.Written() {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": c.Errors[0].Error(),
				})
			}
		}
	}
}
