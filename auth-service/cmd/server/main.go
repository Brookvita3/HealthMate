package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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

	myApp := app.NewApp(&cfg)

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := myApp.Start(); err != nil {
			log.Printf("Application exited with error: %v", err)
		}
		stopChan <- syscall.SIGTERM
	}()

	<-stopChan

	log.Println("OS interrupt signal received. Initiating shutdown...")
	myApp.Shutdown()
	log.Println("Application has been shut down.")
}
