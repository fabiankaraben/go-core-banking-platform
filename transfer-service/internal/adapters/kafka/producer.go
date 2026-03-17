package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer is a Kafka event publisher adapter.
type Producer struct {
	client *kgo.Client
}

// NewProducer creates a new Kafka Producer.
func NewProducer(brokers []string) (*Producer, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return nil, fmt.Errorf("creating kafka producer: %w", err)
	}
	return &Producer{client: client}, nil
}

// Publish sends a message to the specified Kafka topic.
func (p *Producer) Publish(ctx context.Context, topic, key string, payload []byte) error {
	record := &kgo.Record{Topic: topic, Key: []byte(key), Value: payload}
	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("publishing to topic %q: %w", topic, err)
	}
	return nil
}

// Close shuts down the Kafka client.
func (p *Producer) Close() error {
	p.client.Close()
	return nil
}
