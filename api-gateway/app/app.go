package app

import (
	"log"

	"api-gateway/config"
	"api-gateway/internal/kafka"

	"github.com/gin-gonic/gin"
)

type App struct {
	Config        *config.Config
	KafkaProducer kafka.Producer
	JWTSecret     string
	Router        *gin.Engine
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

	return &App{
		Router:        router,
		Config:        cfg,
		KafkaProducer: KafkaProducer,
		JWTSecret:     cfg.JWTSecret,
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
