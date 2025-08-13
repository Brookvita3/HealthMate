package main

import (
	"os"

	"heathhub/app"
	"heathhub/config"
	"heathhub/internal/router"
)

func main() {
	cfg := config.LoadConfig()

	app := app.NewApp(cfg)

	r := router.SetupRouter(app)

	r.Run(":" + os.Getenv("PORT"))
}
