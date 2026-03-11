package app

import (
	"storage-service/config"
	"storage-service/internal/kafka"
	"storage-service/internal/metric"
	"log"
	"net/http"

	postgrePlatform "storage-service/internal/platform/postgres"
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type App struct {
	MetricRepo    metric.MetricRepository
	MetricService metric.Service
	MetricHandler *metric.Handler
	KafkaConsumer *kafka.Consumer
	pgPool        *pgxpool.Pool
	RedisClient   *redis.Client
	Router        *gin.Engine
}

func NewApp(cfg *config.Config) *App {

	pool, err := postgrePlatform.NewTimeScaleDBConnFromURL(cfg.TimeScaleDbURL)
	log.Printf("Postgres URL: %s", config.LoadConfig().TimeScaleDbURL)
	if err != nil {
		log.Fatal("Unable to create connection pool: ", err.Error())
	}
	log.Println("Successfully connected to Postgres.")

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatal("Unable to parse Redis URL: ", err.Error())
	}
	redisClient := redis.NewClient(opt)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis at %s: %v", cfg.RedisURL, err)
	} else {
		log.Println("Successfully connected to Redis.")
	}

	metricRepo := metric.NewRepository(pool, redisClient)
	metricService := metric.NewMetricService(metricRepo)
	metricHandler := metric.NewHandler(metricService)

	kafkaConsumer := kafka.NewConsumer(cfg.KafkaAddr, cfg.KafkaTopic, cfg.KafkaGroupID, metricService)

	router := gin.Default()
	setupRoutes(router, metricHandler)

	return &App{
		MetricRepo:    metricRepo,
		MetricService: metricService,
		MetricHandler: metricHandler,
		KafkaConsumer: kafkaConsumer,
		pgPool:        pool,
		RedisClient:   redisClient,
		Router:        router,
	}
}

func setupRoutes(r *gin.Engine, metricHandler *metric.Handler) {
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

	metrics := r.Group("/metrics")
	{
		metrics.GET("/charts", metricHandler.GetChartData)
	}
}

func (a *App) Start(ctx context.Context, cfg *config.Config, errCh chan error) {
	go a.KafkaConsumer.Start(ctx, errCh)
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
