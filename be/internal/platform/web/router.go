package web

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"healthmate/internal/auth"
	"healthmate/internal/data"
	"healthmate/internal/platform/web/middleware"
	"healthmate/internal/realtime"
	"healthmate/internal/user"
)

func SetupDataRoutes(r *gin.Engine, dataHandler *data.Handler) {
	r.POST("/api/health", dataHandler.SendData)
}

func NewRouter(
	authHandler *auth.Handler,
	dataHandler *data.Handler,
	rtHandler *realtime.Handler,
	userHandler *user.Handler,
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

	// ===== Users routes =====
	userRoutes := apiV1.Group("/users")
	userRoutes.Use(authHandler.AuthMiddleware())
	{
		userRoutes.GET("/me", userHandler.GetProfile)
		userRoutes.PUT("/me", userHandler.UpdateProfile)
		userRoutes.GET("", userHandler.ListUsers) // /api/v1/users?search=john&limit=10&offset=20
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
