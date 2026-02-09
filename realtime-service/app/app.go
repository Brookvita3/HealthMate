package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"realtime-service/config"
	"time"

	"realtime-service/internal/auth"
	"realtime-service/internal/kafka"
	"realtime-service/internal/metric"
	"realtime-service/internal/permission"
	postgresPlatform "realtime-service/internal/platform/postgres"
	"realtime-service/internal/realtime"

	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "realtime-service/docs"
)

type Dependencies struct {
	Config         *config.Config
	PgPool         *pgxpool.Pool
	PermRepo       permission.Repository
	TokenValidator auth.TokenValidator
}

type App struct {
	HttpServer     *http.Server
	RealtimeEngine *RealtimeEngine
	Deps           *Dependencies

	ctx    context.Context
	cancel context.CancelFunc
}

type RealtimeEngine struct {
	Hub      *realtime.Hub
	Consumer *kafka.Consumer
	Producer *kafka.Producer
	metricCh chan *metric.HealthMetric
	ctx      context.Context
}

func NewApp(cfg *config.Config) *App {
	appCtx, appCancel := context.WithCancel(context.Background())

	deps := NewDependencies(cfg)
	realtimeEngine := NewRealtimeEngine(appCtx, deps)
	httpServer := NewHttpServer(deps, realtimeEngine.Hub)

	return &App{
		HttpServer:     httpServer,
		RealtimeEngine: realtimeEngine,
		Deps:           deps,
		ctx:            appCtx,
		cancel:         appCancel,
	}
}

func NewDependencies(cfg *config.Config) *Dependencies {
	pool, err := postgresPlatform.NewPostgreSQLConnFromURL(cfg.PostgreURL)
	log.Printf("Postgres URL: %s", cfg.PostgreURL)
	if err != nil {
		log.Fatal("Unable to create connection pool: ", err.Error())
	}
	log.Println("Successfully connected to Postgres.")
	permRepo := permission.NewRepository(pool)

	tokenValidator := auth.NewJWTValidator(cfg.JWTSecret)

	return &Dependencies{
		PgPool:         pool,
		PermRepo:       permRepo,
		TokenValidator: tokenValidator,
		Config:         cfg,
	}
}

func NewHttpServer(deps *Dependencies, hub *realtime.Hub) *http.Server {
	wsHandler := realtime.NewHandler(deps.TokenValidator, deps.PermRepo, hub)

	mux := http.NewServeMux()
	mux.Handle("/ws", wsHandler)
	mux.Handle("/swagger/", httpSwagger.Handler())

	httpServer := &http.Server{
		Addr:    ":" + deps.Config.HTTPPort,
		Handler: mux,
	}
	return httpServer
}

func NewRealtimeEngine(ctx context.Context, deps *Dependencies) *RealtimeEngine {
	producer := kafka.NewProducer(deps.Config.KafkaAddr, deps.Config.KafkaTopic)
	hub := realtime.NewHub(deps.PermRepo, producer)

	metricCh := make(chan *metric.HealthMetric, 100)

	consumer := kafka.NewConsumer(deps.Config.KafkaAddr, deps.Config.KafkaTopic, deps.Config.KafkaGroupID, metricCh)

	return &RealtimeEngine{
		Hub:      hub,
		Consumer: consumer,
		Producer: producer,
		metricCh: metricCh,
		ctx:      ctx,
	}
}

func (e *RealtimeEngine) Run(errCh chan error) {
	go e.Hub.Run(e.ctx)
	go e.Consumer.Start(e.ctx, errCh)

	log.Println("RealtimeEngine is running and listening for metrics...")
	for {
		select {
		case metric, ok := <-e.metricCh:
			if !ok {
				log.Println("Metric channel closed, engine is stopping.")
				return
			}
			e.Hub.BroadcastMetric(metric)
		case <-e.ctx.Done():
			log.Println("RealtimeEngine received stop signal.")
			return
		}
	}
}

func (a *App) Start() error {
	log.Println("Starting application services...")

	errCh := make(chan error, 1)

	go a.RealtimeEngine.Run(errCh)

	go func() {
		log.Printf("HTTP server listening on %s", a.HttpServer.Addr)
		if err := a.HttpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		log.Printf("A critical error occurred: %v. Triggering shutdown...", err)
		a.Shutdown()
		return err
	case <-a.ctx.Done():
		log.Println("Application context canceled. Starting graceful shutdown.")
		a.Shutdown()
		return nil
	}
}

func (a *App) Shutdown() {
	log.Println("Executing graceful shutdown...")

	a.cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := a.HttpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server gracefully stopped.")
	}

	a.Deps.PgPool.Close()
	log.Println("PostgreSQL connection pool closed.")

	if a.RealtimeEngine.Producer != nil {
		a.RealtimeEngine.Producer.Close()
		log.Println("Kafka producer closed.")
	}

	log.Println("Shutdown complete.")
}
