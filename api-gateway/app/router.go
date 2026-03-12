package app

import (
	"api-gateway/config"
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"
)

func (a *App) SetupRoutes(cfg config.Config) {

	a.Router.Use(middleware.CORSMiddleware())
	a.Router.Any("/auth/*proxyPath",
		handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/auth"))

	protected := a.Router.Group("")
	protected.Use(middleware.JWTAuthMiddleware(a.JWTSecret))
	{
		protected.Any("/user/data",
			handlers.KafkaSendHandler(a.KafkaProducer, cfg.KafkaTopic))

		protected.Any("/groups/*proxyPath",
			handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/groups"))

		protected.Any("/metrics/*proxyPath",
			handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/metrics"))
	}

	a.Router.GET("/gateway/health", handlers.PingHandler)
}
