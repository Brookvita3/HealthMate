package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"realtime-service/internal/metric"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader   *kafka.Reader
	metricCh chan<- *metric.HealthMetric
}

func NewConsumer(brokers, topic, groupID string, metricCh chan<- *metric.HealthMetric) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokers},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: r, metricCh: metricCh}
}

func (c *Consumer) Start(ctx context.Context, errCh chan error) {
	defer c.reader.Close()
	for {
		select {
		case <-ctx.Done():
			log.Println("Kafka consumer stopped")
			return
		default:
			m, err := c.reader.ReadMessage(ctx)
			if err != nil {
				errCh <- fmt.Errorf("kafka read error: %w", err)
				return
			}

			var metric metric.HealthMetric
			if err := json.Unmarshal(m.Value, &metric); err != nil {
				log.Println("Unmarshal message error:", err)
				continue
			}

			log.Printf("Received message: %v", metric)
			select {
			case c.metricCh <- &metric:

			case <-ctx.Done():
				return
			}
		}
	}
}
