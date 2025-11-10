package app

import (
	"auth-service/config"
	"auth-service/internal/auth"
	grpcserver "auth-service/internal/grpc"
	email "auth-service/internal/mail"
	postgrePlatform "auth-service/internal/platform/postgres"
	redisPlatform "auth-service/internal/platform/redis"
	"auth-service/internal/user"
	"auth-service/internal/web/middleware"
	authpb "auth-service/proto/pb"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	redis "github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"
)

type App struct {
	Deps       *Dependencies
	HTTPServer *HTTPServer
	GRPCServer *GRPCServer
}

type Dependencies struct {
	DB              *pgxpool.Pool
	Redis           *redis.Client
	Config          *config.Config
	JWTTokenService *auth.JWTTokenService

	TokenService auth.TokenService
	UserRepo     user.UserRepository
	OTPService   auth.OTPService
	AuthService  auth.Service
}

type HTTPServer struct {
	Router *gin.Engine
	server *http.Server
}

type GRPCServer struct {
	server *grpc.Server
	addr   string
}

func NewGRPCServer(deps *Dependencies) *GRPCServer {
	s := grpc.NewServer()

	authpb.RegisterAuthServiceServer(s, &grpcserver.AuthGRPCServer{
		JwtService: deps.JWTTokenService,
	})
	return &GRPCServer{
		server: s,
		addr:   ":" + deps.Config.GRPCPort,
	}
}

func NewApp(cfg *config.Config) *App {
	deps := NewDependencies(cfg)

	httpServer := NewHTTPServer(deps)
	grpcServer := NewGRPCServer(deps)

	return &App{
		Deps:       deps,
		HTTPServer: httpServer,
		GRPCServer: grpcServer,
	}
}

func NewHTTPServer(deps *Dependencies) *HTTPServer {
	router := gin.Default()

	router.Use(middleware.PrometheusMiddleware())

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group(deps.Config.APIPrefix)

	// ===== Auth routes =====
	authHandler := auth.NewHandler(deps.AuthService, deps.TokenService)
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/google", authHandler.GoogleLogin)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/otp/verify", authHandler.VerifyAccount)
		authGroup.POST("/otp/resend", authHandler.ResendOTP)

		authGroup.POST("/app", authHandler.AppLogin)

		authGroup.Use(authHandler.AuthMiddleware())
		authGroup.POST("/logout", authHandler.LogOut)
		authGroup.POST("/password", authHandler.SetPassword)
	}

	srv := &http.Server{
		Addr:    ":" + deps.Config.HTTPPort,
		Handler: router,
	}

	return &HTTPServer{
		Router: router,
		server: srv,
	}
}

func NewDependencies(cfg *config.Config) *Dependencies {

	db, err := postgrePlatform.NewPostgreSQLConnFromURL(cfg.PostgreURL)
	if err != nil {
		log.Fatalf("Failed to connect Postgres: %v", err)
	}

	redisClient, err := redisPlatform.NewRedisClientFromURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect Redis: %v", err)
	}

	repo := user.NewRepository(db)
	cache := redisPlatform.NewCacheWrapper(redisClient)
	jwtService := auth.NewJWTTokenService(cfg.JWTSecret, cache)
	otpService := auth.NewRedisOTPService(cache)
	tokenService := auth.NewJWTTokenService(cfg.JWTSecret, cache)
	emailService := email.NewGmailEmailService(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername,
		cfg.SMTPAppPassword, cfg.SMTPSenderName,
	)
	googleVerifier := auth.NewGoogleTokenVerifierImpl(cfg.GoogleClientID)

	authService := auth.NewAuthService(repo, jwtService, otpService, emailService, googleVerifier)

	return &Dependencies{
		DB:              db,
		Redis:           redisClient,
		Config:          cfg,
		UserRepo:        repo,
		JWTTokenService: jwtService,
		TokenService:    tokenService,
		OTPService:      otpService,
		AuthService:     authService,
	}
}

func (s *HTTPServer) Start(addr string) error {
	log.Printf("HTTP server listening on %s", addr)
	return s.Router.Run(addr)
}

func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	log.Printf("gRPC server listening on %s", s.addr)
	return s.server.Serve(lis)
}

func (s *HTTPServer) Stop() error {
	log.Println("Stopping HTTP server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *GRPCServer) Stop() {
	log.Println("Stopping gRPC server...")
	s.server.GracefulStop()
}
