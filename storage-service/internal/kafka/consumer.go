package kafka

import (
	"storage-service/internal/common"
	"storage-service/internal/metric"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader        *kafka.Reader
	metricServcie metric.Service
}

func NewConsumer(brokers, topic, groupID string, metricServcie metric.Service) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokers},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: r, metricServcie: metricServcie}
}

func (c *Consumer) Start(ctx context.Context, errCh chan error) {
	defer c.reader.Close()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Kafka consumer stopped")
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

			if err := c.metricServcie.RecordMetric(ctx, metric); err != nil {
				var businessErr *common.BusinessError
				if errors.As(err, &businessErr) {
					log.Printf("Process message warning (Business Error): %s (User: %s, Metric: %s)", businessErr.Message, metric.UserID, metric.Type)
				} else {
					log.Printf("Process message system error: %v (User: %s, Metric: %s)", err, metric.UserID, metric.Type)
				}
			}
		}
	}
}
