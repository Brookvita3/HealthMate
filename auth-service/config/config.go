package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret       string
	RedisURL        string
	RedisUsername   string
	RedisPassword   string
	Port            string
	GoogleClientID  string
	PostgreURL      string
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPAppPassword string
	SMTPSenderName  string
	APIPrefix       string
}

func LoadConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	SMTP_PORT, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		log.Fatalf("Error converting string to integer: %v", err)
	}

	config := Config{
		JWTSecret:       os.Getenv("JWT_SECRET"),
		RedisURL:        os.Getenv("REDIS_URL"),
		RedisUsername:   os.Getenv("REDIS_USERNAME"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		Port:            os.Getenv("PORT"),
		GoogleClientID:  os.Getenv("GOOGLE_CLIENT_ID"),
		PostgreURL:      os.Getenv("POSTGRES_URL"),
		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPPort:        SMTP_PORT,
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPAppPassword: os.Getenv("SMTP_APP_PASSWORD"),
		SMTPSenderName:  os.Getenv("SMTP_SENDER_NAME"),
		APIPrefix:       os.Getenv("API_PREFIX"),
	}

	return config
}
