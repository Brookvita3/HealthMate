package main

import (
	"api-gateway/app"
	"api-gateway/config"
	"context"
	"fmt"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := config.LoadConfig()

	errCh := make(chan error, 1)

	app := app.NewApp(&cfg)

	app.Start(ctx, &cfg, errCh)

	select {
	case err := <-errCh:
		fmt.Println("Kafka consumer error:", err)
		cancel()
	case <-ctx.Done():
		fmt.Println("Service stopped")
	}
}
