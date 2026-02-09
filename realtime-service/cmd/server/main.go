package main

import (
	"log"

	"os"
	"os/signal"
	"realtime-service/app"
	"realtime-service/config"
	"syscall"

	_ "realtime-service/docs"
)

// @title HealthMate Realtime API
// @version 1.0
// @description This is the API for Realtime Service of HealthMate application.
// @host localhost:5001
// @BasePath /
// @schemes http https

// @SecurityDefinitions.apikey BearerAuth
// @In query
// @Name token
// @Description Token should be passed as a query parameter for WebSocket connection.
func main() {
	cfg := config.LoadConfig()

	application := app.NewApp(&cfg)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := application.Start(); err != nil {
			log.Printf("Application exited with error: %v", err)
		}
	}()

	<-stopChan

	log.Println("OS interrupt signal received. Initiating shutdown...")
	application.Shutdown()

	log.Println("Application has been shut down.")
}
