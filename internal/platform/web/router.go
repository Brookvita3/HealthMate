package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"healthmate/internal/auth"
	"healthmate/internal/data"
	"healthmate/internal/platform/web/middleware"
)

func SetupDataRoutes(r *gin.Engine, dataHandler *data.Handler) {
	r.POST("/api/health", dataHandler.SendData)
}

func NewRouter(
	authHandler *auth.Handler,
	dataHandler *data.Handler,
) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())

	apiV1 := r.Group("/api/v1")

	// ===== Auth routes =====
	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/google", authHandler.GoogleLogin)
		authGroup.POST("/refresh", authHandler.RefreshToken)

		authGroup.Use(authHandler.AuthMiddleware())
		authGroup.POST("/logout", authHandler.LogOut)
	}

	// ===== Profile routes =====
	profileGroup := apiV1.Group("/profile")
	{
		profileGroup.Use(authHandler.AuthMiddleware())
		profileGroup.GET("/", func(c *gin.Context) {
			email := c.GetString("email")
			userID := c.GetString("userID")
			c.JSON(http.StatusOK, gin.H{"email": email, "userID": userID})
		})
	}

	// ===== Health routes =====
	healthGroup := apiV1.Group("/health")
	{
		healthGroup.Use(authHandler.AuthMiddleware())
		healthGroup.POST("/", dataHandler.SendData)
	}

	return r

}
