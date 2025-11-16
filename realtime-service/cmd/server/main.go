package main

import (
	"log"

	"os"
	"os/signal"
	"realtime-service/app"
	"realtime-service/config"
	"syscall"
)

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
