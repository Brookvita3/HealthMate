package app

import (
	"log"

	"api-gateway/config"
	"api-gateway/internal/kafka"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Config        *config.Config
	KafkaProducer kafka.Producer
	JWTSecret     string
	Router        *gin.Engine
	RedisClient   *redis.Client
}

func NewApp(cfg *config.Config) *App {
	router := gin.Default()
	// Tránh 307/301 (trailing slash / fixed path) trước khi middleware CORS xử lý OPTIONS —
	// trình duyệt báo "Redirect is not allowed for a preflight request".
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	KafkaProducer := kafka.NewKafkaProducer([]string{cfg.KafkaBrokerURL})

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	return &App{
		Router:        router,
		Config:        cfg,
		KafkaProducer: KafkaProducer,
		JWTSecret:     cfg.JWTSecret,
		RedisClient:   rdb,
	}
}

func (a *App) Start(addr string) {
	log.Printf("Server starting on %s", addr)
	if err := a.Router.Run(addr); err != nil {
		log.Fatalf("Failed to start api-gateway: %v", err)
	}
}

func (a *App) Shutdown() {
	log.Println("Shutting down api-gateway...")
}
