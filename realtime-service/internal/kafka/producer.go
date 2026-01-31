package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"realtime-service/internal/metric"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) PublishMetric(ctx context.Context, m *metric.HealthMetric) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(m.UserID),
		Value: data,
	})

	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	log.Printf("Successfully published metric to Kafka for user: %s", m.UserID)
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
