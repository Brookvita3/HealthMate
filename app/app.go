package app

import (
	"heathhub/config"

	"github.com/redis/go-redis/v9"

	"heathhub/internal/handler"
	"heathhub/internal/repository"
	"heathhub/internal/service"
	"heathhub/pkg/auth"
)

type App struct {
	TokenService *auth.TokenService
	AuthHandler  *handler.AuthHandler
	DataHandler  *handler.DataHandler
}

func NewApp(cfg config.Config) *App {

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})

	userRepo := repository.NewUserRepository("users.json")

	tokenService := auth.NewTokenService(cfg.JWTSecret, redisClient)

	authService := service.NewAuthService(userRepo, tokenService, cfg.GoogleClientID)

	authHandler := handler.NewAuthHandler(authService)

	dataHandler := handler.NewDataHandler()

	return &App{
		TokenService: tokenService,
		AuthHandler:  authHandler,
		DataHandler:  dataHandler,
	}
}
