package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/domain"
	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/ports"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// TransferService implements ports.TransferService.
type TransferService struct {
	repo       ports.TransferRepository
	outbox     ports.OutboxRepository
	idempotency ports.IdempotencyStore
	logger     *zap.Logger
}

// NewTransferService creates a new TransferService.
func NewTransferService(repo ports.TransferRepository, outbox ports.OutboxRepository, idempotency ports.IdempotencyStore, logger *zap.Logger) *TransferService {
	return &TransferService{
		repo:        repo,
		outbox:      outbox,
		idempotency: idempotency,
		logger:      logger,
	}
}

// CreateTransfer initiates a new money transfer using the Saga pattern.
func (s *TransferService) CreateTransfer(ctx context.Context, idempotencyKey string, sourceID, destID uuid.UUID, amount decimal.Decimal, currency string) (*domain.Transfer, error) {
	// Check idempotency store first.
	if cached, err := s.idempotency.Get(ctx, idempotencyKey); err == nil && cached != nil {
		var t domain.Transfer
		if err := json.Unmarshal(cached, &t); err == nil {
			return &t, nil
		}
	}

	// Check DB for existing transfer with same idempotency key.
	existing, err := s.repo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil && !errors.Is(err, domain.ErrTransferNotFound) {
		return nil, fmt.Errorf("checking idempotency key: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	transfer, err := domain.NewTransfer(idempotencyKey, sourceID, destID, amount, currency)
	if err != nil {
		return nil, fmt.Errorf("creating transfer domain object: %w", err)
	}

	if err := s.repo.Create(ctx, transfer); err != nil {
		return nil, fmt.Errorf("persisting transfer: %w", err)
	}

	event := domain.TransferRequestedEvent{
		TransferID:      transfer.ID.String(),
		SourceAccountID: sourceID.String(),
		DestAccountID:   destID.String(),
		Amount:          amount.String(),
		Currency:        currency,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshalling TransferRequestedEvent: %w", err)
	}
	outboxEvt := &domain.OutboxEvent{
		ID:      uuid.New().String(),
		Topic:   domain.TopicTransferRequested,
		Key:     transfer.ID.String(),
		Payload: payload,
	}
	if err := s.outbox.SaveEvent(ctx, outboxEvt); err != nil {
		return nil, fmt.Errorf("saving outbox event: %w", err)
	}

	// Cache in Redis for fast idempotency replay.
	if serialized, err := json.Marshal(transfer); err == nil {
		_ = s.idempotency.Set(ctx, idempotencyKey, serialized)
	}

	s.logger.Info("transfer created", zap.String("transfer_id", transfer.ID.String()))
	return transfer, nil
}

// GetTransfer retrieves a transfer by its ID.
func (s *TransferService) GetTransfer(ctx context.Context, id uuid.UUID) (*domain.Transfer, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching transfer: %w", err)
	}
	return t, nil
}

// HandleTransferCompleted processes a TransferCompletedEvent from Kafka.
func (s *TransferService) HandleTransferCompleted(ctx context.Context, event domain.TransferCompletedEvent) error {
	id, err := uuid.Parse(event.TransferID)
	if err != nil {
		return fmt.Errorf("parsing transfer id: %w", err)
	}
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching transfer for completion: %w", err)
	}
	t.MarkCompleted()
	if err := s.repo.UpdateWithVersion(ctx, t); err != nil {
		return fmt.Errorf("updating transfer to completed: %w", err)
	}
	s.logger.Info("transfer marked completed", zap.String("transfer_id", event.TransferID))
	return nil
}

// HandleTransferFailed processes a TransferFailedEvent from Kafka.
func (s *TransferService) HandleTransferFailed(ctx context.Context, event domain.TransferFailedEvent) error {
	id, err := uuid.Parse(event.TransferID)
	if err != nil {
		return fmt.Errorf("parsing transfer id: %w", err)
	}
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching transfer for failure: %w", err)
	}
	t.MarkFailed(event.Reason)
	if err := s.repo.UpdateWithVersion(ctx, t); err != nil {
		return fmt.Errorf("updating transfer to failed: %w", err)
	}
	s.logger.Info("transfer marked failed",
		zap.String("transfer_id", event.TransferID),
		zap.String("reason", event.Reason),
	)
	return nil
}
