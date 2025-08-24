package web

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"healthmate/internal/auth"
	"healthmate/internal/data"
	"healthmate/internal/platform/web/middleware"
	"healthmate/internal/realtime"
)

func SetupDataRoutes(r *gin.Engine, dataHandler *data.Handler) {
	r.POST("/api/health", dataHandler.SendData)
}

func NewRouter(
	authHandler *auth.Handler,
	dataHandler *data.Handler,
	rtHandler *realtime.Handler,
) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(gin.Recovery())
	r.RedirectTrailingSlash = true
	r.Use(middleware.ErrorHandler())

	apiV1 := r.Group("/api/v1")

	// ===== Auth routes =====
	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/google", authHandler.GoogleLogin)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/app", authHandler.AppLogin)

		authGroup.Use(authHandler.AuthMiddleware())
		authGroup.POST("/logout", authHandler.LogOut)
		authGroup.POST("/password", authHandler.SetPassword)
	}

	// ===== Profile routes =====
	profileGroup := apiV1.Group("/profile")
	{
		profileGroup.Use(authHandler.AuthMiddleware())
		profileGroup.GET("", func(c *gin.Context) {
			email := c.GetString("email")
			userId := c.GetString("userId")
			c.JSON(http.StatusOK, gin.H{"email": email, "userID": userId})
		})
	}

	// ===== Health routes =====
	healthGroup := apiV1.Group("/health")
	{
		healthGroup.Use(authHandler.AuthMiddleware())
		healthGroup.POST("", dataHandler.SendData)
	}

	// ===== Websocket routes =====
	wsGroup := apiV1.Group("/ws")
	{
		wsGroup.Use(authHandler.AuthMiddleware())
		wsGroup.GET("", rtHandler.ServeWs)
	}

	return r

}
