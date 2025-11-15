package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"auth-service/app"
	"auth-service/config"
)

// @title HealthMate API
// @version 1.0
// @description This is the API for the HealthMate application.

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

// @SecurityDefinitions.apikey BearerAuth
// @In header
// @Name Authorization
// @Description Type "Bearer" followed by a space and JWT token.
func main() {
	cfg := config.LoadConfig()

	runDBMigration(cfg.PostgreURL)

	myApp := app.NewApp(&cfg)

	go func() {
		if err := myApp.HTTPServer.Start(":" + cfg.HTTPPort); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	go func() {
		if err := myApp.GRPCServer.Start(); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := myApp.HTTPServer.Stop(ctx); err != nil {
		log.Printf("Error stopping HTTP server: %v", err)
	}
	myApp.GRPCServer.Stop()
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
