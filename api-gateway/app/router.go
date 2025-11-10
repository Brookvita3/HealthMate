package app

import (
	"api-gateway/config"
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"
)

func (a *App) SetupRoutes(cfg config.Config) {

	a.Router.Any("/auth/*proxyPath",
		handlers.ReverseProxy(cfg.AuthHTTPURL+cfg.APIPrefix+"/auth"))

	protected := a.Router.Group("")
	protected.Use(middleware.AuthMiddleware(a.AuthClient))
	{
		protected.Any("/user/data",
			handlers.KafkaSendHandler(a.KafkaProducer, cfg.KafkaTopic))
	}
	a.Router.Any("/ws", handlers.WebSocketProxy(cfg.RealtimeHTTPURL))
}
