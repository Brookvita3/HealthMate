package main

import (
	"storage-service/app"
	"storage-service/config"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "storage-service/docs"
)

// @title HealthMate Storage API
// @version 1.0
// @description This is the API for Storage Service of HealthMate application.

// @host localhost:5003
// @BasePath /api/v1
// @schemes http https

// @SecurityDefinitions.apikey BearerAuth
// @In header
// @Name Authorization
// @Description Type "Bearer" followed by a space and JWT token.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.LoadConfig()

	runDBMigration(cfg.TimeScaleDbURL)

	errCh := make(chan error, 1)

	application := app.NewApp(&cfg)

	application.Start(ctx, &cfg, errCh)

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Printf("Critical error occurred: %v. Triggering shutdown...", err)
		cancel()
		os.Exit(1)
	case <-sigCh:
		fmt.Println("Shutting down...")
	case <-ctx.Done():
		fmt.Println("Context cancelled")
	}
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
	log.Printf("Storage DB version: %d, dirty: %v, error: %v", version, dirty, err)

	err = m.Up()

	if err != nil {
		if err == migrate.ErrNoChange {
			log.Println("migrate up finished without error (no change).")
		} else {
			log.Printf("migrate up finished with error: %v", err)
			log.Fatalf("failed to run migrate up: %v", err)
		}
	} else {
		log.Println("Storage DB migrated successfully")
	}
}
