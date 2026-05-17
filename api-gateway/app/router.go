package app

import (
	"api-gateway/config"
	"api-gateway/internal/handlers"
	"api-gateway/internal/middleware"
)

func (a *App) SetupRoutes(cfg config.Config) {

	a.Router.Use(middleware.CORSMiddleware())

	// Public routes with rate limiting (uses API Key or IP since sub is not set)
	authGroup := a.Router.Group("/auth")
	authGroup.Use(middleware.RateLimitMiddleware(a.RedisClient, cfg.RateLimitLimit, cfg.RateLimitWindow))
	authGroup.Any("/*proxyPath", handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/auth"))

	protected := a.Router.Group("")
	protected.Use(middleware.JWTAuthMiddleware(a.JWTSecret))
	protected.Use(middleware.RateLimitMiddleware(a.RedisClient, cfg.RateLimitLimit, cfg.RateLimitWindow))
	{
		// Users
		protected.Any("/users", handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/users"))
		protected.Any("/users/*proxyPath", handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/users"))

		// Groups
		protected.Any("/groups", handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/groups"))
		protected.Any("/groups/*proxyPath", handlers.ReverseProxy(cfg.AuthHTTPURL, cfg.APIPrefix+"/groups"))

		// Metrics
		protected.Any("/metrics", handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/metrics"))
		protected.Any("/metrics/*proxyPath", handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/metrics"))

		// Medications
		protected.Any("/medications", handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/medications"))
		protected.Any("/medications/*proxyPath", handlers.ReverseProxy(cfg.StorageHTTPURL, cfg.APIPrefix+"/medications"))

		// OCR (ocr-service)
		protected.Any("/ocr/*proxyPath", handlers.ReverseProxy(cfg.OCRHTTPURL, "/ocr"))
	}

	a.Router.GET("/health", handlers.PingHandler)
	a.Router.GET("/ready", handlers.ReadyHandler(a.RedisClient))
}
