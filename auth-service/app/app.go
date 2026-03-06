package app

import (
	"auth-service/config"
	"auth-service/internal/auth"
	"auth-service/internal/group"
	email "auth-service/internal/mail"
	"auth-service/internal/member"
	"auth-service/internal/permission"
	postgrePlatform "auth-service/internal/platform/postgres"
	redisPlatform "auth-service/internal/platform/redis"
	"auth-service/internal/user"
	"auth-service/internal/validation"
	"auth-service/internal/web/middleware"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	redis "github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type App struct {
	Deps       *Dependencies
	HTTPServer *HTTPServer
}

type Dependencies struct {
	DB              *pgxpool.Pool
	Redis           *redis.Client
	Config          *config.Config
	JWTTokenService *auth.JWTTokenService

	TokenService      auth.TokenService
	UserRepo          user.UserRepository
	GroupRepo         group.GroupRepository
	MemberRepo        member.MemberRepository
	OTPService        auth.OTPService
	AuthService       auth.Service
	GroupService      group.Service
	MemberService     member.Service
	PermissionService permission.Service
	Validator         *validation.Validator
}

type HTTPServer struct {
	Router *gin.Engine
	server *http.Server
}

func NewApp(cfg *config.Config) *App {
	deps := NewDependencies(cfg)

	httpServer := NewHTTPServer(deps)

	return &App{
		Deps:       deps,
		HTTPServer: httpServer,
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

	// ===== Group Middleware =====
	groupMiddleware := middleware.NewGroupMiddleware(deps.Validator)

	// ===== Group routes =====
	groupHandler := group.NewHandler(deps.GroupService, deps.MemberService, deps.PermissionService)
	groupGroup := api.Group("/groups")
	groupGroup.Use(authHandler.AuthMiddleware())
	{
		// Routes that don't need group validation (no :id param)
		groupGroup.POST("", groupHandler.CreateGroup)
		groupGroup.GET("", groupHandler.ListMyGroups)
		groupGroup.GET("/metric-types", groupHandler.ListMetricTypes)
		groupGroup.GET("/invitations", groupHandler.ListInvitations)

		// Routes that need group validation (have :id param)
		// Create a subgroup with group validation middleware
		groupWithID := groupGroup.Group("/:id")
		groupWithID.Use(groupMiddleware.ValidateGroupExists())
		{
			groupWithID.GET("", groupHandler.GetGroup)
			groupWithID.PUT("", groupHandler.UpdateGroup)
			groupWithID.DELETE("", groupHandler.DeleteGroup)

			groupWithID.POST("/members", groupHandler.InviteMember)
			groupWithID.GET("/members", groupHandler.GetMembers)
			groupWithID.PUT("/members/me", groupHandler.UpdateMyMemberStatus)
			groupWithID.DELETE("/members/me", groupHandler.LeaveGroup)

			groupWithID.PUT("/owner", groupHandler.TransferOwnership)
			groupWithID.POST("/permissions", groupHandler.SetPermission)
			groupWithID.PUT("/permissions", groupHandler.UpdatePermissions)
			groupWithID.GET("/permissions", groupHandler.GetPermissions)

			// Routes that need member validation
			groupWithID.DELETE("/members/:member_id", groupMiddleware.ValidateMemberExists(), groupHandler.RemoveMember)
		}
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

	userRepo := user.NewRepository(db)
	groupRepo := group.NewPostgresRepository(db)
	memberRepo := member.NewPostgresRepository(db)
	permissionRepo := permission.NewPostgresRepository(db)

	cache := redisPlatform.NewCacheWrapper(redisClient)
	jwtService := auth.NewJWTTokenService(cfg.JWTSecret, cache)
	otpService := auth.NewRedisOTPService(cache)
	tokenService := auth.NewJWTTokenService(cfg.JWTSecret, cache)
	emailService := email.NewGmailEmailService(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername,
		cfg.SMTPAppPassword, cfg.SMTPSenderName,
	)
	googleVerifier := auth.NewGoogleTokenVerifierImpl(cfg.GoogleClientID)

	authService := auth.NewAuthService(userRepo, jwtService, otpService, emailService, googleVerifier)

	// Create Validator with repository adapters
	memberChecker := validation.NewMemberCheckerAdapter(memberRepo)
	validator := validation.NewValidator(groupRepo, userRepo, memberChecker)

	return &Dependencies{
		DB:                db,
		Redis:             redisClient,
		Config:            cfg,
		UserRepo:          userRepo,
		GroupRepo:         groupRepo,
		MemberRepo:        memberRepo,
		JWTTokenService:   jwtService,
		TokenService:      tokenService,
		OTPService:        otpService,
		AuthService:       authService,
		GroupService:      group.NewService(groupRepo, userRepo),
		MemberService:     member.NewService(memberRepo, userRepo),
		PermissionService: permission.NewService(permissionRepo),
		Validator:         validator,
	}
}

func (s *HTTPServer) Start() error {
	log.Printf("HTTP server listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (a *App) Shutdown() {
	log.Println("Executing graceful shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := a.HTTPServer.Stop(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	a.Deps.DB.Close()
	log.Println("PostgreSQL connection pool closed.")

	a.Deps.Redis.Close()
	log.Println("Redis connection pool closed.")

	log.Println("Shutdown complete.")
}

func (a *App) Start() error {
	errChan := make(chan error, 2)

	go func() { errChan <- a.HTTPServer.Start() }()

	return <-errChan
}
