package main

import (
	"api-gateway/app"
	"api-gateway/config"
)

func main() {

	cfg := config.LoadConfig()

	app := app.NewApp(&cfg)

	app.SetupRoutes(cfg)

	app.Start(":" + cfg.Port)

	defer app.Shutdown()
}
