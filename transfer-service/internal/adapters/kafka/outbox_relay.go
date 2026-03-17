package kafka

import (
	"context"
	"time"

	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/ports"
	"go.uber.org/zap"
)

// OutboxRelay polls the outbox table and publishes pending events to Kafka.
type OutboxRelay struct {
	outbox    ports.OutboxRepository
	publisher ports.EventPublisher
	logger    *zap.Logger
	interval  time.Duration
}

// NewOutboxRelay creates a new OutboxRelay.
func NewOutboxRelay(outbox ports.OutboxRepository, publisher ports.EventPublisher, logger *zap.Logger) *OutboxRelay {
	return &OutboxRelay{outbox: outbox, publisher: publisher, logger: logger, interval: 2 * time.Second}
}

// Start begins the relay loop. It blocks until ctx is cancelled.
func (r *OutboxRelay) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	r.logger.Info("transfer outbox relay started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.relay(ctx)
		}
	}
}

func (r *OutboxRelay) relay(ctx context.Context) {
	events, err := r.outbox.GetUnpublished(ctx, 50)
	if err != nil {
		r.logger.Error("fetching unpublished outbox events", zap.Error(err))
		return
	}
	for _, e := range events {
		if err := r.publisher.Publish(ctx, e.Topic, e.Key, e.Payload); err != nil {
			r.logger.Error("publishing outbox event", zap.String("id", e.ID), zap.Error(err))
			continue
		}
		if err := r.outbox.MarkPublished(ctx, e.ID); err != nil {
			r.logger.Error("marking outbox event published", zap.String("id", e.ID), zap.Error(err))
		}
	}
}
