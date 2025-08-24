package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"healthmate/app"
	"healthmate/config"
	"healthmate/internal/auth"
	"healthmate/internal/data"
	"healthmate/internal/platform/cache"
	"healthmate/internal/platform/web"
	"healthmate/internal/realtime"
	"healthmate/internal/user"
	"healthmate/pkg/jwtauth"
)

func main() {
	cfg := config.LoadConfig()

	redisClient, err := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, 0)
	log.Printf("Redis URL: %s", config.LoadConfig().RedisAddr)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("Successfully connected to Redis.")
	defer redisClient.Close()

	rtManager := realtime.NewManager()
	go rtManager.Run()

	pool, err := pgxpool.New(context.Background(), config.LoadConfig().PostgreURL)
	log.Printf("Postgres URL: %s", config.LoadConfig().PostgreURL)
	if err != nil {
		log.Fatal("Unable to create connection pool: ", err.Error())
	}
	log.Println("Successfully connected to Postgres.")
	defer pool.Close()

	runDBMigration(config.LoadConfig().PostgreURL)

	userRepo := user.NewRepository(pool)

	tokenService := jwtauth.NewTokenService(cfg.JWTSecret, redisClient)

	authService := auth.NewAuthService(userRepo, tokenService, cfg.GoogleClientID)

	userService := user.NewUserService(userRepo)

	authHandler := auth.NewHandler(authService)

	dataHandler := data.NewDataHandler()

	rtHandler := realtime.NewHandler(rtManager)

	userHandler := user.NewHandler(userService)

	router := web.NewRouter(authHandler, dataHandler, rtHandler, userHandler)

	application := app.NewApp(router)

	application.Start(":" + cfg.Port)
}

func runDBMigration(databaseUrl string) {
	absPath, err := filepath.Abs("migration")
	if err != nil {
		log.Fatalf("cannot find absolute path: %v", err)
	}

	log.Printf("Using migration path: %s", absPath)

	migrationPath := "file://" + filepath.ToSlash(absPath)

	m, err := migrate.New(migrationPath, databaseUrl)
	if err != nil {
		log.Fatalf("cannot create new migrate instance: %v", err)
	}

	version, dirty, err := m.Version()
	log.Printf("DB version: %d, dirty: %v, error: %v", version, dirty, err)

	err = m.Up()

	if err != nil {
		log.Printf("migrate up finished with error: %v", err)
	} else {
		log.Println("migrate up finished without error.")
	}

	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migrate up: %v", err)
	}

	log.Println("DB migrated successfully")
}
