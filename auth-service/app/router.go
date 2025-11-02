package app

import (
	_ "auth-service/docs"
	"auth-service/internal/web/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

func (a *App) SetupRoutes() {

	apiV1 := a.Router.Group("/api/v1")

	a.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	a.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	a.Router.Use(middleware.PrometheusMiddleware())

	// ===== Auth routes =====
	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/google", a.AuthHandler.GoogleLogin)
		authGroup.POST("/refresh", a.AuthHandler.RefreshToken)
		authGroup.POST("/register", a.AuthHandler.Register)
		authGroup.POST("/otp/verify", a.AuthHandler.VerifyAccount)
		authGroup.POST("/otp/resend", a.AuthHandler.ResendOTP)

		authGroup.POST("/app", a.AuthHandler.AppLogin)

		authGroup.Use(a.AuthHandler.AuthMiddleware())
		authGroup.POST("/logout", a.AuthHandler.LogOut)
		authGroup.POST("/password", a.AuthHandler.SetPassword)
	}

}
