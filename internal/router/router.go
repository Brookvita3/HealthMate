package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heathhub/app"
	"heathhub/internal/handler"
	"heathhub/internal/middleware"
	"heathhub/pkg/auth"
)

func SetupRouter(app *app.App) *gin.Engine {

	r := gin.Default()

	r.Use(middleware.ErrorHandler())

	SetupAuthRoutes(r, app.AuthHandler, app.TokenService)

	return r
}

func SetupAuthRoutes(r *gin.Engine, authHandler *handler.AuthHandler, tokenService *auth.TokenService) {
	r.POST("/auth/google", authHandler.GoogleLogin)
	r.POST("/auth/refresh", authHandler.RefreshAccessToken)

	authRoutes := r.Group("/auth")
	authRoutes.Use(middleware.AuthMiddleware(tokenService))
	{
		authRoutes.GET("/profile", func(c *gin.Context) {
			email := c.GetString("email")
			c.JSON(http.StatusOK, gin.H{"email": email})
		})
	}
}
