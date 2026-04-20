package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret       string
	AuthHTTPURL     string
	RealtimeHTTPURL string
	OCRHTTPURL      string
	Port            string
	KafkaBrokerURL  string
	KafkaTopic      string
	StorageHTTPURL  string
	OCRHTTPURL      string
	APIPrefix       string
}

func LoadConfig() Config {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	config := Config{
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AuthHTTPURL:     os.Getenv("AUTH_HTTP_URL"),
		Port:            os.Getenv("PORT"),
		KafkaBrokerURL:  os.Getenv("KAFKA_BROKER_URL"),
		KafkaTopic:      os.Getenv("KAFKA_TOPIC"),
		APIPrefix:       os.Getenv("API_PREFIX"),
		RealtimeHTTPURL: os.Getenv("REALTIME_HTTP_URL"),
		StorageHTTPURL:  os.Getenv("STORAGE_HTTP_URL"),
		OCRHTTPURL:      os.Getenv("OCR_HTTP_URL"),
	}

	return config
}
