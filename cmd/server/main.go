package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"healthmate/app"
	"healthmate/config"
	"healthmate/internal/auth"
	"healthmate/internal/data"
	"healthmate/internal/platform/cache"
	"healthmate/internal/platform/web"
	"healthmate/internal/realtime"
	"healthmate/pkg/jwtauth"
)

func main() {
	cfg := config.LoadConfig()

	redisClient, err := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, 0)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("Successfully connected to Redis.")
	defer redisClient.Close()

	rtManager := realtime.NewManager()
	go rtManager.Run()

	pool, err := pgxpool.New(context.Background(), config.LoadConfig().PostgreURL)
	log.Printf("Postgres URL: %s", config.LoadConfig().PostgreURL)
	if err != nil {
		log.Fatal("Unable to create connection pool: ", err.Error())
	}
	log.Println("Successfully connected to Postgres.")
	defer pool.Close()

	userRepo := auth.NewRepository(pool)

	tokenService := jwtauth.NewTokenService(cfg.JWTSecret, redisClient)

	authService := auth.NewAuthService(userRepo, tokenService, cfg.GoogleClientID)

	authHandler := auth.NewHandler(authService)

	dataHandler := data.NewDataHandler()

	rtHandler := realtime.NewHandler(rtManager)

	router := web.NewRouter(authHandler, dataHandler, rtHandler)

	application := app.NewApp(router)

	application.Start(":" + cfg.Port)
}
