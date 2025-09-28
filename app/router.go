package app

import (
	_ "healthmate/docs"

	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

func (a *App) SetupRoutes() {
	apiV1 := a.Router.Group("/api/v1")

	a.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ===== Auth routes =====
	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/google", a.AuthHandler.GoogleLogin)
		authGroup.POST("/refresh", a.AuthHandler.RefreshToken)
		authGroup.POST("/register", a.AuthHandler.Register)
		authGroup.POST("/app", a.AuthHandler.AppLogin)

		authGroup.Use(a.AuthHandler.AuthMiddleware())
		authGroup.POST("/logout", a.AuthHandler.LogOut)
		authGroup.POST("/password", a.AuthHandler.SetPassword)
	}

}
