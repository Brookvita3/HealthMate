package app

import (
	"github.com/redis/go-redis/v9"
	"heathhub/config"

	"heathhub/internal/handler"
	"heathhub/internal/repository"
	"heathhub/internal/service"
	"heathhub/pkg/auth"
)

type App struct {
	TokenService *auth.TokenService
	AuthHandler  *handler.AuthHandler
}

func NewApp(cfg config.Config) *App {

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})

	userRepo := repository.NewUserRepository("users.json")

	tokenService := auth.NewTokenService(cfg.JWTSecret, redisClient)

	authService := service.NewAuthService(userRepo, tokenService)

	authHandler := handler.NewAuthHandler(authService)

	return &App{
		TokenService: tokenService,
		AuthHandler:  authHandler,
	}
}
