package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"api-gateway/config"
	"api-gateway/internal/kafka"
	"api-gateway/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Config        *config.Config
	KafkaProducer kafka.Producer
	JWTSecret     string
	Router        *gin.Engine
	RedisClient   *redis.Client
	Server        *http.Server
}

func NewApp(cfg *config.Config) *App {
	router := gin.Default()
	// Tránh 307/301 (trailing slash / fixed path) trước khi middleware CORS xử lý OPTIONS —
	// trình duyệt báo "Redirect is not allowed for a preflight request".
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	router.Use(middleware.PrometheusMiddleware())
	router.GET("/prometheus-metrics", gin.WrapH(promhttp.Handler()))

	KafkaProducer := kafka.NewKafkaProducer([]string{cfg.KafkaBrokerURL})

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	rdb := redis.NewClient(opt)

	return &App{
		Router:        router,
		Config:        cfg,
		KafkaProducer: KafkaProducer,
		JWTSecret:     cfg.JWTSecret,
		RedisClient:   rdb,
		Server: &http.Server{
			Addr:    ":" + cfg.Port,
			Handler: router,
		},
	}
}

func (a *App) Start() error {
	log.Printf("Server starting on %s", a.Server.Addr)
	return a.Server.ListenAndServe()
}

func (a *App) Shutdown() {
	log.Println("Executing graceful shutdown for api-gateway...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if a.RedisClient != nil {
		a.RedisClient.Close()
		log.Println("Redis connection closed.")
	}

	log.Println("Shutdown complete.")
}
