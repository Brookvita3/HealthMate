package app

import (
	"context"
	"healthmate/config"
	"healthmate/internal/auth"
	email "healthmate/internal/mail"
	redisPlatform "healthmate/internal/platform/redis"
	"healthmate/internal/user"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

type App struct {
	UserRepo user.UserRepository

	AuthService  auth.Service
	TokenService auth.TokenService
	OTPService   auth.OTPService

	AuthHandler auth.Handler

	Router      *gin.Engine
	pgPool      *pgxpool.Pool
	redisClient *redis.Client
}

func NewApp(cfg *config.Config) *App {

	redisClient, err := redisPlatform.NewRedisClientFromURL(cfg.RedisURL)
	log.Printf("Redis URL: %s", config.LoadConfig().RedisURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("Successfully connected to Redis.")

	pool, err := pgxpool.New(context.Background(), config.LoadConfig().PostgreURL)
	log.Printf("Postgres URL: %s", config.LoadConfig().PostgreURL)
	if err != nil {
		log.Fatal("Unable to create connection pool: ", err.Error())
	}
	log.Println("Successfully connected to Postgres.")

	userRepo := user.NewRepository(pool)

	redisCache := redisPlatform.NewCacheWrapper(redisClient)

	GmailService := email.NewGmailEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPAppPassword, cfg.SMTPSenderName)

	JWTTokenService := auth.NewJWTTokenService(cfg.JWTSecret, redisCache)
	RedisOTPService := auth.NewRedisOTPService(redisCache)
	GoogleTokenVerifierImpl := auth.NewGoogleTokenVerifierImpl(cfg.GoogleClientID)
	authService := auth.NewAuthService(userRepo, JWTTokenService, RedisOTPService, GmailService, GoogleTokenVerifierImpl)

	authHandler := auth.NewHandler(authService, JWTTokenService)

	router := gin.Default()

	return &App{
		UserRepo:     userRepo,
		AuthService:  authService,
		TokenService: JWTTokenService,
		AuthHandler:  *authHandler,
		Router:       router,
		pgPool:       pool,
		redisClient:  redisClient,
	}
}

func (a *App) Start(addr string) {
	log.Printf("Server starting on %s", addr)
	if err := a.Router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
func (a *App) Shutdown() {
	log.Println("Shutting down server...")
	a.pgPool.Close()
	log.Println("Postgres connection closed.")
	a.redisClient.Close()
	log.Println("Redis connection closed.")
}
