package app

import (
	"log"

	"api-gateway/config"
	"api-gateway/internal/kafka"
	authpb "api-gateway/proto/pb"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	Config        *config.Config
	KafkaProducer kafka.Producer
	AuthClient    authpb.AuthServiceClient
	Router        *gin.Engine
}

func NewApp(cfg *config.Config) *App {
	router := gin.Default()

	KafkaProducer := kafka.NewKafkaProducer([]string{cfg.KafkaBrokerURL})

	grpcConn, err := grpc.NewClient(cfg.AuthGRPCURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect auth service: %v", err)
	}
	authClient := authpb.NewAuthServiceClient(grpcConn)

	return &App{
		Router:        router,
		Config:        cfg,
		KafkaProducer: KafkaProducer,
		AuthClient:    authClient,
	}
}

func (a *App) Start(addr string) {
	log.Printf("Server starting on %s", addr)
	if err := a.Router.Run(addr); err != nil {
		log.Fatalf("Failed to start api-gateway: %v", err)
	}
}

func (a *App) Shutdown() {
	log.Println("Shutting down api-gateway...")
}
