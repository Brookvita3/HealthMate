package app

import (
	"api-gateway/config"
	"api-gateway/internal/kafka"
	"api-gateway/internal/metric"
	"log"

	postgrePlatform "api-gateway/internal/platform/postgres"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	MetricRepo    metric.MetricRepository
	MetricService metric.Service
	KafkaConsumer *kafka.Consumer
	pgPool        *pgxpool.Pool
}

func NewApp(cfg *config.Config) *App {

	pool, err := postgrePlatform.NewTimeScaleDBConnFromURL(cfg.TimeScaleDbURL)
	log.Printf("Postgres URL: %s", config.LoadConfig().TimeScaleDbURL)
	if err != nil {
		log.Fatal("Unable to create connection pool: ", err.Error())
	}
	log.Println("Successfully connected to Postgres.")

	metricRepo := metric.NewRepository(pool)
	metricService := metric.NewMetricService(metricRepo)

	kafkaConsumer := kafka.NewConsumer(cfg.KafkaAddr, cfg.KafkaTopic, cfg.KafkaGroupID, metricService)

	return &App{
		MetricRepo:    metricRepo,
		MetricService: metricService,
		KafkaConsumer: kafkaConsumer,
		pgPool:        pool,
	}
}

func (a *App) Start(ctx context.Context, cfg *config.Config, errCh chan error) {
	go a.KafkaConsumer.Start(ctx, errCh)
	fmt.Println("Storage service started with Kafka consumer at address", cfg.KafkaAddr)
}
