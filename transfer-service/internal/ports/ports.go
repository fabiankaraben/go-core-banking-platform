package ports

import (
	"context"

	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── Inbound Ports ─────────────────────────────────────────────────────────

// TransferService is the primary business logic port.
type TransferService interface {
	CreateTransfer(ctx context.Context, idempotencyKey string, sourceID, destID uuid.UUID, amount decimal.Decimal, currency string) (*domain.Transfer, error)
	GetTransfer(ctx context.Context, id uuid.UUID) (*domain.Transfer, error)
	HandleTransferCompleted(ctx context.Context, event domain.TransferCompletedEvent) error
	HandleTransferFailed(ctx context.Context, event domain.TransferFailedEvent) error
}

// ─── Outbound Ports ─────────────────────────────────────────────────────────

// TransferRepository is the persistence port for transfers.
type TransferRepository interface {
	Create(ctx context.Context, transfer *domain.Transfer) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Transfer, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transfer, error)
	UpdateWithVersion(ctx context.Context, transfer *domain.Transfer) error
}

// OutboxRepository is the persistence port for the transactional outbox.
type OutboxRepository interface {
	SaveEvent(ctx context.Context, event *domain.OutboxEvent) error
	GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
}

// IdempotencyStore is the port for Redis-backed idempotency checks.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
}

// EventPublisher is the port for publishing events to the message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload []byte) error
	Close() error
}
