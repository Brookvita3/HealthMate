package app

import (
	"log"

	"github.com/gin-gonic/gin"
)

type App struct {
	Router *gin.Engine
}

func NewApp(router *gin.Engine) *App {
	return &App{
		Router: router,
	}
}

func (a *App) Start(addr string) {
	log.Printf("Server starting on %s", addr)
	if err := a.Router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
