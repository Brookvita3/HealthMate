package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	KafkaAddr      string
	Port           string
	TimeScaleDbURL string
	KafkaTopic     string
	KafkaGroupID   string
}

func LoadConfig() Config {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	config := Config{
		KafkaAddr:      os.Getenv("KAFKA_ADDR"),
		Port:           os.Getenv("PORT"),
		TimeScaleDbURL: os.Getenv("TIMESCALEDB_URL"),
		KafkaTopic:     os.Getenv("KAFKA_TOPIC"),
		KafkaGroupID:   os.Getenv("KAFKA_GROUP_ID"),
	}

	return config
}
