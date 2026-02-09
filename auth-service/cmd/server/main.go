package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"auth-service/app"
	"auth-service/config"
	_ "auth-service/docs"
)

// @title HealthMate API
// @version 1.0
// @description This is the API for Auth Service of HealthMate application.

// @host localhost:5000
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

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := myApp.Start(); err != nil {
			log.Printf("Application exited with error: %v", err)
		}
	}()

	<-stopChan

	log.Println("OS interrupt signal received. Initiating shutdown...")
	myApp.Shutdown()
	log.Println("Application has been shut down.")
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
