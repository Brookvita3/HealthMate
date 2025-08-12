package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heathhub/internal/handler"
	"heathhub/internal/middleware"
)

func SetupRouter(authHandler *handler.AuthHandler) *gin.Engine {

	r := gin.Default()

	r.Use(middleware.ErrorHandler())

	r.POST("/auth/google", authHandler.GoogleLogin)

	authRoutes := r.Group("/")
	authRoutes.Use(middleware.AuthMiddleware())
	{
		authRoutes.GET("/profile", func(c *gin.Context) {
			email := c.GetString("email")
			c.JSON(http.StatusOK, gin.H{"email": email})
		})
	}
	return r
}
