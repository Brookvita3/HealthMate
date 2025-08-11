package main

import (
	"heathhub/config"
	"heathhub/internal/handler"
	"heathhub/internal/repository"
	"heathhub/internal/router"
	"heathhub/internal/service"
	"os"
)

func main() {
	config.LoadConfig()

	userRepo := repository.NewUserRepository("users.json")
	authService := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authService)

	r := router.SetupRouter(authHandler)
	r.Run(":" + os.Getenv("PORT"))
}
