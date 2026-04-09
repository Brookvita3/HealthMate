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
		protected.Any("/users/*proxyPath",
			handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/users"))

		protected.Any("/groups/*proxyPath",
			handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/groups"))

		protected.Any("/metrics/*proxyPath",
			handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/metrics"))

		protected.Any("/medications/*proxyPath",
			handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/medications"))
	}

	a.Router.GET("/gateway/health", handlers.PingHandler)
}
