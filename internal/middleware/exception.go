package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {

	// handle panic
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal Server Error",
				})
				c.Abort()
			}
		}()

		c.Next()

		// if have error, handle in here
		if len(c.Errors) > 0 {
			log.Println("Handler errors:", c.Errors)
			c.JSON(-1, gin.H{
				"error": c.Errors[0].Error(),
			})
			c.Abort()
		}
	}
}
