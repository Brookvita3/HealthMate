package app

import (
	"log"
	"net/http"
	"storage-service/config"
	"storage-service/internal/kafka"
	"storage-service/internal/metric"
	"storage-service/internal/middleware"

	"context"
	"fmt"
	"net"
	"time"
	postgrePlatform "storage-service/internal/platform/postgres"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type App struct {
	MetricRepo    metric.MetricRepository
	MetricService metric.Service
	MetricHandler *metric.Handler
	KafkaConsumer *kafka.Consumer
	pgPool        *pgxpool.Pool
	Router        *gin.Engine
}

func NewApp(cfg *config.Config) *App {

	var pool *pgxpool.Pool
	var err error
	log.Printf("Postgres URL: %s", cfg.TimeScaleDbURL)

	pool, err = postgrePlatform.NewTimeScaleDBConnFromURL(cfg.TimeScaleDbURL)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	log.Println("Successfully connected to Postgres.")

	metricRepo := metric.NewRepository(pool)
	metricService := metric.NewMetricService(metricRepo)
	metricHandler := metric.NewHandler(metricService)

	kafkaConsumer := kafka.NewConsumer(cfg.KafkaAddr, cfg.KafkaTopic, cfg.KafkaGroupID, metricService)

	router := gin.Default()
	setupRoutes(router, metricHandler, cfg.JWTSecret, cfg.APIPrefix)

	return &App{
		MetricRepo:    metricRepo,
		MetricService: metricService,
		MetricHandler: metricHandler,
		KafkaConsumer: kafkaConsumer,
		pgPool:        pool,
		Router:        router,
	}
}

func setupRoutes(r *gin.Engine, metricHandler *metric.Handler, jwtSecret string, apiPrefix string) {
	// @Summary Health Check
	// @Description Check if the service is up
	// @Tags health
	// @Produce json
	// @Success 200 {object} map[string]string
	// @Router /health [get]
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group(apiPrefix)
	metrics := api.Group("/metrics")
	metrics.Use(middleware.JWTAuthMiddleware(jwtSecret))
	{
		metrics.GET("/charts", metricHandler.GetChartData)
	}
}

func (a *App) Start(ctx context.Context, cfg *config.Config, errCh chan error) {
	go func() {
		conn, err := net.DialTimeout("tcp", cfg.KafkaAddr, 3*time.Second)
		if err != nil {
			errCh <- fmt.Errorf("failed to connect to Kafka brokers at %s: %w", cfg.KafkaAddr, err)
			return
		}
		conn.Close()
		log.Println("Successfully connected to Kafka brokers.")

		a.KafkaConsumer.Start(ctx, errCh)
	}()

	log.Printf("Storage service started with Kafka consumer at address %s", cfg.KafkaAddr)

	go func() {
		port := cfg.Port
		if port == "" {
			port = "5003"
		}
		log.Printf("Starting HTTP server on port %s", port)
		if err := a.Router.Run(":" + port); err != nil {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()
}
