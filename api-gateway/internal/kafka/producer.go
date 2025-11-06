package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	Send(ctx context.Context, topic string, key, value []byte) error
	Close() error
}

func NewKafkaProducer(brokers []string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

type KafkaProducer struct {
	writer *kafka.Writer
}

func (p *KafkaProducer) Send(ctx context.Context, topic string, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
