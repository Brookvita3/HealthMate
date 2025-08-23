package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret      string
	RedisAddr      string
	RedisPassword  string
	Port           string
	GoogleClientID string
	PostgreURL     string
}

func LoadConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	config := Config{
		JWTSecret:      os.Getenv("JWT_SECRET"),
		RedisAddr:      os.Getenv("REDIS_ADDR"),
		RedisPassword:  os.Getenv("REDIS_PASSWORD"),
		Port:           os.Getenv("PORT"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		PostgreURL:     os.Getenv("POSTGRES_URL"),
	}

	return config
}
