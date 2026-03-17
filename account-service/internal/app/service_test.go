package app_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/app"
	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Mock AccountRepository ────────────────────────────────────────────────

type mockAccountRepo struct{ mock.Mock }

func (m *mockAccountRepo) Create(ctx context.Context, a *domain.Account) error {
	return m.Called(ctx, a).Error(0)
}
func (m *mockAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}
func (m *mockAccountRepo) UpdateWithVersion(ctx context.Context, a *domain.Account) error {
	return m.Called(ctx, a).Error(0)
}

// ─── Mock OutboxRepository ────────────────────────────────────────────────

type mockOutboxRepo struct{ mock.Mock }

func (m *mockOutboxRepo) SaveEvent(ctx context.Context, e *domain.OutboxEvent) error {
	return m.Called(ctx, e).Error(0)
}
func (m *mockOutboxRepo) GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*domain.OutboxEvent), args.Error(1)
}
func (m *mockOutboxRepo) MarkPublished(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// ─── Tests ─────────────────────────────────────────────────────────────────

func TestAccountService_CreateAccount(t *testing.T) {
	repo := new(mockAccountRepo)
	outbox := new(mockOutboxRepo)
	logger := zap.NewNop()
	svc := app.NewAccountService(repo, outbox, logger)

	customerID := uuid.New()
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Account")).Return(nil)

	account, err := svc.CreateAccount(context.Background(), customerID, "USD", decimal.NewFromFloat(500))
	require.NoError(t, err)
	assert.Equal(t, customerID, account.CustomerID)
	assert.Equal(t, "500", account.Balance.String())
	repo.AssertExpectations(t)
}

func TestAccountService_ProcessTransferRequested_Success(t *testing.T) {
	repo := new(mockAccountRepo)
	outbox := new(mockOutboxRepo)
	logger := zap.NewNop()
	svc := app.NewAccountService(repo, outbox, logger)

	sourceID := uuid.New()
	destID := uuid.New()
	transferID := uuid.New().String()

	source, _ := domain.NewAccount(uuid.New(), "USD", decimal.NewFromFloat(1000))
	source.ID = sourceID
	dest, _ := domain.NewAccount(uuid.New(), "USD", decimal.Zero)
	dest.ID = destID

	repo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	repo.On("GetByID", mock.Anything, destID).Return(dest, nil)
	repo.On("UpdateWithVersion", mock.Anything, mock.AnythingOfType("*domain.Account")).Return(nil).Twice()
	outbox.On("SaveEvent", mock.Anything, mock.MatchedBy(func(e *domain.OutboxEvent) bool {
		return e.Topic == domain.TopicTransferCompleted
	})).Return(nil)

	event := domain.TransferRequestedEvent{
		TransferID:      transferID,
		SourceAccountID: sourceID.String(),
		DestAccountID:   destID.String(),
		Amount:          "250.00",
		Currency:        "USD",
	}
	err := svc.ProcessTransferRequested(context.Background(), event)
	require.NoError(t, err)

	assert.Equal(t, "750", source.Balance.String())
	assert.Equal(t, "250", dest.Balance.String())
	repo.AssertExpectations(t)
	outbox.AssertExpectations(t)
}

func TestAccountService_ProcessTransferRequested_InsufficientFunds(t *testing.T) {
	repo := new(mockAccountRepo)
	outbox := new(mockOutboxRepo)
	logger := zap.NewNop()
	svc := app.NewAccountService(repo, outbox, logger)

	sourceID := uuid.New()
	destID := uuid.New()

	source, _ := domain.NewAccount(uuid.New(), "USD", decimal.NewFromFloat(50))
	source.ID = sourceID
	dest, _ := domain.NewAccount(uuid.New(), "USD", decimal.Zero)
	dest.ID = destID

	repo.On("GetByID", mock.Anything, sourceID).Return(source, nil)
	repo.On("GetByID", mock.Anything, destID).Return(dest, nil)
	outbox.On("SaveEvent", mock.Anything, mock.MatchedBy(func(e *domain.OutboxEvent) bool {
		if e.Topic != domain.TopicTransferFailed {
			return false
		}
		var evt domain.TransferFailedEvent
		_ = json.Unmarshal(e.Payload, &evt)
		return evt.Reason == domain.ErrInsufficientFunds.Error()
	})).Return(nil)

	event := domain.TransferRequestedEvent{
		TransferID:      uuid.New().String(),
		SourceAccountID: sourceID.String(),
		DestAccountID:   destID.String(),
		Amount:          "500.00",
		Currency:        "USD",
	}
	err := svc.ProcessTransferRequested(context.Background(), event)
	require.NoError(t, err)

	repo.AssertExpectations(t)
	outbox.AssertExpectations(t)
}

func TestAccountService_ProcessTransferRequested_SourceNotFound(t *testing.T) {
	repo := new(mockAccountRepo)
	outbox := new(mockOutboxRepo)
	logger := zap.NewNop()
	svc := app.NewAccountService(repo, outbox, logger)

	sourceID := uuid.New()

	repo.On("GetByID", mock.Anything, sourceID).Return(nil, domain.ErrAccountNotFound)
	outbox.On("SaveEvent", mock.Anything, mock.MatchedBy(func(e *domain.OutboxEvent) bool {
		return e.Topic == domain.TopicTransferFailed
	})).Return(nil)

	event := domain.TransferRequestedEvent{
		TransferID:      uuid.New().String(),
		SourceAccountID: sourceID.String(),
		DestAccountID:   uuid.New().String(),
		Amount:          "100.00",
		Currency:        "USD",
	}
	err := svc.ProcessTransferRequested(context.Background(), event)
	require.NoError(t, err)
	outbox.AssertExpectations(t)
}
