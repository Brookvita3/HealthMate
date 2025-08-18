package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"healthmate/app"
	"healthmate/internal/handler"
	"healthmate/internal/middleware"
	"healthmate/pkg/auth"
)

func SetupRouter(app *app.App) *gin.Engine {

	r := gin.Default()

	r.Use(middleware.ErrorHandler())

	SetupAuthRoutes(r, app.AuthHandler, app.TokenService)
	SetupDataRoutes(r, app.DataHandler)

	return r
}

func SetupAuthRoutes(r *gin.Engine, authHandler *handler.AuthHandler, tokenService *auth.TokenService) {
	r.POST("/auth/google", authHandler.GoogleLogin)
	r.POST("/auth/refresh", authHandler.RefreshToken)

	authRoutes := r.Group("/auth")
	authRoutes.Use(middleware.AuthMiddleware(tokenService))
	{
		authRoutes.GET("/profile", func(c *gin.Context) {
			email := c.GetString("email")
			c.JSON(http.StatusOK, gin.H{"email": email})
		})
		authRoutes.POST("/logout", authHandler.LogOut)
	}
}

func SetupDataRoutes(r *gin.Engine, dataHandler *handler.DataHandler) {
	r.POST("/api/health", dataHandler.SendData)
}
