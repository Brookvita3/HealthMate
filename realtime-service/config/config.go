package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret      string
	HTTPPort       string
	KafkaBrokerURL string
	KafkaTopic     string
	KafkaAddr      string
	PostgreURL     string
	KafkaGroupID   string
}

func LoadConfig() Config {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	config := Config{
		JWTSecret:    os.Getenv("JWT_SECRET"),
		HTTPPort:     os.Getenv("HTTP_PORT"),
		KafkaTopic:   os.Getenv("KAFKA_TOPIC"),
		KafkaGroupID: os.Getenv("KAFKA_GROUP_ID"),
		KafkaAddr:    os.Getenv("KAFKA_ADDR"),
		PostgreURL:   os.Getenv("POSTGRES_URL"),
	}

	return config
}
