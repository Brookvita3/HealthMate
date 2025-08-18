package main

import (
	"os"

	"healthmate/app"
	"healthmate/config"
	"healthmate/internal/router"
)

func main() {
	cfg := config.LoadConfig()

	app := app.NewApp(cfg)

	r := router.SetupRouter(app)

	r.Run(":" + os.Getenv("PORT"))
}
