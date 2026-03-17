package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/domain"
	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/ports"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// AccountService implements the ports.AccountService inbound port.
type AccountService struct {
	repo    ports.AccountRepository
	outbox  ports.OutboxRepository
	logger  *zap.Logger
}

// NewAccountService creates a new AccountService.
func NewAccountService(repo ports.AccountRepository, outbox ports.OutboxRepository, logger *zap.Logger) *AccountService {
	return &AccountService{
		repo:   repo,
		outbox: outbox,
		logger: logger,
	}
}

// CreateAccount creates a new bank account.
func (s *AccountService) CreateAccount(ctx context.Context, customerID uuid.UUID, currency string, initialBalance decimal.Decimal) (*domain.Account, error) {
	account, err := domain.NewAccount(customerID, currency, initialBalance)
	if err != nil {
		return nil, fmt.Errorf("creating account domain object: %w", err)
	}
	if err := s.repo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("persisting account: %w", err)
	}
	s.logger.Info("account created", zap.String("account_id", account.ID.String()))
	return account, nil
}

// GetAccount retrieves an account by ID.
func (s *AccountService) GetAccount(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching account: %w", err)
	}
	return account, nil
}

// ProcessTransferRequested handles a TransferRequestedEvent from Kafka.
// It debits the source account and credits the destination account within a logical sequence,
// then writes the outcome event to the outbox (Transactional Outbox Pattern).
func (s *AccountService) ProcessTransferRequested(ctx context.Context, event domain.TransferRequestedEvent) error {
	amount, err := decimal.NewFromString(event.Amount)
	if err != nil {
		return fmt.Errorf("parsing amount %q: %w", event.Amount, err)
	}

	sourceID, err := uuid.Parse(event.SourceAccountID)
	if err != nil {
		return fmt.Errorf("parsing source account id: %w", err)
	}
	destID, err := uuid.Parse(event.DestAccountID)
	if err != nil {
		return fmt.Errorf("parsing dest account id: %w", err)
	}

	source, err := s.repo.GetByID(ctx, sourceID)
	if err != nil {
		return s.publishFailure(ctx, event.TransferID, "source account not found", event.TraceID)
	}
	dest, err := s.repo.GetByID(ctx, destID)
	if err != nil {
		return s.publishFailure(ctx, event.TransferID, "destination account not found", event.TraceID)
	}

	if err := source.Debit(amount, event.Currency); err != nil {
		s.logger.Warn("debit failed", zap.String("transfer_id", event.TransferID), zap.Error(err))
		return s.publishFailure(ctx, event.TransferID, err.Error(), event.TraceID)
	}
	if err := dest.Credit(amount, event.Currency); err != nil {
		s.logger.Warn("credit failed", zap.String("transfer_id", event.TransferID), zap.Error(err))
		return s.publishFailure(ctx, event.TransferID, err.Error(), event.TraceID)
	}

	if err := s.repo.UpdateWithVersion(ctx, source); err != nil {
		return fmt.Errorf("updating source account: %w", err)
	}
	if err := s.repo.UpdateWithVersion(ctx, dest); err != nil {
		return fmt.Errorf("updating dest account: %w", err)
	}

	completedEvt := domain.TransferCompletedEvent{
		TransferID: event.TransferID,
		TraceID:    event.TraceID,
	}
	payload, err := json.Marshal(completedEvt)
	if err != nil {
		return fmt.Errorf("marshalling completed event: %w", err)
	}
	outboxEvt := &domain.OutboxEvent{
		ID:      uuid.New().String(),
		Topic:   domain.TopicTransferCompleted,
		Key:     event.TransferID,
		Payload: payload,
	}
	if err := s.outbox.SaveEvent(ctx, outboxEvt); err != nil {
		return fmt.Errorf("saving completed outbox event: %w", err)
	}

	s.logger.Info("transfer processed successfully", zap.String("transfer_id", event.TransferID))
	return nil
}

func (s *AccountService) publishFailure(ctx context.Context, transferID, reason, traceID string) error {
	failedEvt := domain.TransferFailedEvent{
		TransferID: transferID,
		Reason:     reason,
		TraceID:    traceID,
	}
	payload, err := json.Marshal(failedEvt)
	if err != nil {
		return fmt.Errorf("marshalling failed event: %w", err)
	}
	outboxEvt := &domain.OutboxEvent{
		ID:      uuid.New().String(),
		Topic:   domain.TopicTransferFailed,
		Key:     transferID,
		Payload: payload,
	}
	return s.outbox.SaveEvent(ctx, outboxEvt)
}
