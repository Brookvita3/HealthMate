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
	RedisURL       string
	JWTSecret      string
	KafkaTopic     string
	KafkaGroupID   string
	APIPrefix      string
	ModelPath      string // path to readiness_model.onnx
	OnnxLibPath    string // path to libonnxruntime.so / onnxruntime.dll
}

func LoadConfig() Config {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	config := Config{
		KafkaAddr:      os.Getenv("KAFKA_ADDR"),
		Port:           os.Getenv("PORT"),
		TimeScaleDbURL: os.Getenv("TIMESCALEDB_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		KafkaTopic:     os.Getenv("KAFKA_TOPIC"),
		KafkaGroupID:   os.Getenv("KAFKA_GROUP_ID"),
		APIPrefix:      os.Getenv("API_PREFIX"),
		ModelPath:      os.Getenv("MODEL_PATH"),
		OnnxLibPath:    os.Getenv("ONNX_LIB_PATH"),
	}

	return config
}
