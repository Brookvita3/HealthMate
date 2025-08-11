package router

import (
	"github.com/gin-gonic/gin"

	"heathhub/internal/handler"
	"heathhub/internal/middleware"
)

func SetupRouter(authHandler *handler.AuthHandler) *gin.Engine {
	r := gin.Default()
	r.POST("/auth/google", authHandler.GoogleLogin)
	r.Use(middleware.ErrorHandler())
	return r
}
