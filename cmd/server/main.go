package main

import (
	"log"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"healthmate/app"
	"healthmate/config"
)

func main() {
	cfg := config.LoadConfig()

	runDBMigration(cfg.PostgreURL)

	application := app.NewApp(&cfg)

	defer application.Shutdown()

	application.SetupRoutes()

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
