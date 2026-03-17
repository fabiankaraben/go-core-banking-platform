package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/domain"
	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/ports"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// Consumer listens to Kafka topics and dispatches events to the application service.
type Consumer struct {
	client  *kgo.Client
	service ports.AccountService
	logger  *zap.Logger
}

// NewConsumer creates a new Kafka Consumer.
func NewConsumer(brokers []string, groupID string, service ports.AccountService, logger *zap.Logger) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(domain.TopicTransferRequested),
	)
	if err != nil {
		return nil, fmt.Errorf("creating kafka consumer client: %w", err)
	}
	return &Consumer{client: client, service: service, logger: logger}, nil
}

// Start begins the consumption loop. It blocks until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	c.logger.Info("account-service kafka consumer started")
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			c.logger.Info("kafka consumer context cancelled, stopping")
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
	case domain.TopicTransferRequested:
		var event domain.TransferRequestedEvent
		if err := json.Unmarshal(record.Value, &event); err != nil {
			c.logger.Error("unmarshalling TransferRequestedEvent", zap.Error(err))
			return
		}
		if err := c.service.ProcessTransferRequested(ctx, event); err != nil {
			c.logger.Error("processing TransferRequestedEvent",
				zap.String("transfer_id", event.TransferID),
				zap.Error(err),
			)
		}
	default:
		c.logger.Warn("received message on unexpected topic", zap.String("topic", record.Topic))
	}
}
