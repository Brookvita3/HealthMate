package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"api-gateway/app"
	"api-gateway/config"
)

func main() {
	cfg := config.LoadConfig()

	myApp := app.NewApp(&cfg)

	myApp.SetupRoutes(cfg)

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
