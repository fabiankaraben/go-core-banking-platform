package ports

import (
	"context"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── Inbound Ports (driving) ───────────────────────────────────────────────

// AccountService is the primary port for account business logic.
type AccountService interface {
	CreateAccount(ctx context.Context, customerID uuid.UUID, currency string, initialBalance decimal.Decimal) (*domain.Account, error)
	GetAccount(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	ProcessTransferRequested(ctx context.Context, event domain.TransferRequestedEvent) error
}

// ─── Outbound Ports (driven) ───────────────────────────────────────────────

// AccountRepository is the persistence port for accounts.
type AccountRepository interface {
	Create(ctx context.Context, account *domain.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	UpdateWithVersion(ctx context.Context, account *domain.Account) error
}

// OutboxRepository is the persistence port for the transactional outbox.
type OutboxRepository interface {
	SaveEvent(ctx context.Context, event *domain.OutboxEvent) error
	GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
}

// EventPublisher is the port for publishing events to the message broker.
type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload []byte) error
	Close() error
}
