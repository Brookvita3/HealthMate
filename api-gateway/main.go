package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	authServiceURL = "http://localhost:4000" // Auth service
	kafkaBroker    = "localhost:9092"
	kafkaTopic     = "user_data"
)

var kafkaWriter *kafka.Writer

func main() {
	kafkaWriter = &kafka.Writer{
		Addr:         kafka.TCP(kafkaBroker),
		Topic:        kafkaTopic,
		RequiredAcks: kafka.RequireAll,
		Balancer:     &kafka.LeastBytes{},
	}

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/data", dataHandler)

	log.Println("API Gateway running on port 3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

// /login -> forward sang Auth Service
func loginHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Post(authServiceURL+"/login", "application/json", r.Body)
	if err != nil {
		http.Error(w, "Auth service error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// /data -> verify token, push data vào Kafka
func dataHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	verifyReq, _ := json.Marshal(map[string]string{"token": token})
	resp, err := http.Post(authServiceURL+"/verify", "application/json", bytes.NewReader(verifyReq))
	if err != nil {
		http.Error(w, "Auth service error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var verifyRes map[string]bool
	json.NewDecoder(resp.Body).Decode(&verifyRes)
	if !verifyRes["valid"] {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// đọc data từ body và gửi vào Kafka
	data, _ := io.ReadAll(r.Body)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = kafkaWriter.WriteMessages(ctx, kafka.Message{Value: data})
	if err != nil {
		http.Error(w, "Kafka error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"queued"}`))
}
