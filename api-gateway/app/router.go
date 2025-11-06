package app

import (
	"api-gateway/config"
	"api-gateway/internal/handlers"
)

func (a *App) SetupRoutes(cfg config.Config) {

	api := a.Router.Group("")
	api.Any("/auth/*proxyPath", handlers.ReverseProxy(a.Config.AuthServiceURL+cfg.APIPrefix+"/auth"))
	api.Any("/user/data", handlers.KafkaSendHandler(a.KafkaProducer, cfg.KafkaTopic))
}
