package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/domain"
	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/ports"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// Consumer listens to Kafka topics and dispatches events to the application service.
type Consumer struct {
	client  *kgo.Client
	service ports.TransferService
	logger  *zap.Logger
}

// NewConsumer creates a new Kafka Consumer for the transfer-service.
func NewConsumer(brokers []string, groupID string, service ports.TransferService, logger *zap.Logger) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(domain.TopicTransferCompleted, domain.TopicTransferFailed),
	)
	if err != nil {
		return nil, fmt.Errorf("creating kafka consumer: %w", err)
	}
	return &Consumer{client: client, service: service, logger: logger}, nil
}

// Start begins the consumption loop. It blocks until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	c.logger.Info("transfer-service kafka consumer started")
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				c.logger.Error("kafka fetch error", zap.Error(e.Err))
			}
			continue
		}
		fetches.EachRecord(func(record *kgo.Record) {
			c.handleRecord(ctx, record)
		})
	}
}

// Close shuts down the Kafka client.
func (c *Consumer) Close() {
	c.client.Close()
}

func (c *Consumer) handleRecord(ctx context.Context, record *kgo.Record) {
	switch record.Topic {
	case domain.TopicTransferCompleted:
		var event domain.TransferCompletedEvent
		if err := json.Unmarshal(record.Value, &event); err != nil {
			c.logger.Error("unmarshalling TransferCompletedEvent", zap.Error(err))
			return
		}
		if err := c.service.HandleTransferCompleted(ctx, event); err != nil {
			c.logger.Error("handling TransferCompletedEvent", zap.String("transfer_id", event.TransferID), zap.Error(err))
		}
	case domain.TopicTransferFailed:
		var event domain.TransferFailedEvent
		if err := json.Unmarshal(record.Value, &event); err != nil {
			c.logger.Error("unmarshalling TransferFailedEvent", zap.Error(err))
			return
		}
		if err := c.service.HandleTransferFailed(ctx, event); err != nil {
			c.logger.Error("handling TransferFailedEvent", zap.String("transfer_id", event.TransferID), zap.Error(err))
		}
	default:
		c.logger.Warn("received message on unexpected topic", zap.String("topic", record.Topic))
	}
}
