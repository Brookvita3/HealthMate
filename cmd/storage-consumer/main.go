package main

import (
	"context"
	"github.com/segmentio/kafka-go"
	"log"
)

// Cấu hình cơ bản cho consumer
const (
	kafkaBroker = "localhost:9092"
	kafkaTopic  = "health_metrics"
	// Đặt một Group ID tạm thời để test
	// Quan trọng: Nó phải khác với các group ID chính thức sau này
	consumerGroup = "temp-test-group"
)

func main() {
	// Cấu hình Reader
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBroker},
		Topic:   kafkaTopic,
		GroupID: consumerGroup,
		// Bắt đầu đọc từ message cũ nhất nếu đây là group mới
		StartOffset: kafka.FirstOffset,
	})
	defer r.Close()

	log.Printf("--- KAFKA TEMP CONSUMER ---")
	log.Printf("Đang lắng nghe topic '%s'...", kafkaTopic)
	log.Printf("Hãy gửi request đến API Gateway của bạn...")

	// Vòng lặp vô tận để đọc message
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("LỖI KHI ĐỌC MESSAGE: %v", err)
			break // Thoát nếu có lỗi nghiêm trọng
		}

		// In ra message nhận được
		log.Printf("--- MESSAGE NHẬN ĐƯỢC ---")
		log.Printf("Topic: %s | Partition: %d | Offset: %d", m.Topic, m.Partition, m.Offset)
		log.Printf("Key: %s", string(m.Key))
		log.Printf("Value: %s", string(m.Value))
		log.Printf("--------------------------")
	}
}
