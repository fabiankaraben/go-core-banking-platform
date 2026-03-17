package app_test

import (
	"context"
	"testing"

	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/app"
	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Mocks ──────────────────────────────────────────────────────────────────

type mockTransferRepo struct{ mock.Mock }

func (m *mockTransferRepo) Create(ctx context.Context, t *domain.Transfer) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTransferRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transfer, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Transfer), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockTransferRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transfer, error) {
	args := m.Called(ctx, key)
	if v := args.Get(0); v != nil {
		return v.(*domain.Transfer), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockTransferRepo) UpdateWithVersion(ctx context.Context, t *domain.Transfer) error {
	return m.Called(ctx, t).Error(0)
}

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

type mockIdempotencyStore struct{ mock.Mock }

func (m *mockIdempotencyStore) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if v := args.Get(0); v != nil {
		return v.([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockIdempotencyStore) Set(ctx context.Context, key string, value []byte) error {
	return m.Called(ctx, key, value).Error(0)
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestTransferService_CreateTransfer_Success(t *testing.T) {
	repo := new(mockTransferRepo)
	outbox := new(mockOutboxRepo)
	idem := new(mockIdempotencyStore)
	logger := zap.NewNop()

	svc := app.NewTransferService(repo, outbox, idem, logger)

	src, dst := uuid.New(), uuid.New()
	amount := decimal.NewFromFloat(250)

	idem.On("Get", mock.Anything, "key-1").Return(nil, assert.AnError)
	repo.On("GetByIdempotencyKey", mock.Anything, "key-1").Return(nil, domain.ErrTransferNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Transfer")).Return(nil)
	outbox.On("SaveEvent", mock.Anything, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil)
	idem.On("Set", mock.Anything, "key-1", mock.Anything).Return(nil)

	tr, err := svc.CreateTransfer(context.Background(), "key-1", src, dst, amount, "USD")
	require.NoError(t, err)
	assert.Equal(t, domain.TransferStatusPending, tr.Status)
	assert.Equal(t, "250", tr.Amount.String())

	repo.AssertExpectations(t)
	outbox.AssertExpectations(t)
	idem.AssertExpectations(t)
}

func TestTransferService_CreateTransfer_IdempotencyReplay(t *testing.T) {
	repo := new(mockTransferRepo)
	outbox := new(mockOutboxRepo)
	idem := new(mockIdempotencyStore)
	logger := zap.NewNop()

	svc := app.NewTransferService(repo, outbox, idem, logger)

	src, dst := uuid.New(), uuid.New()
	amount := decimal.NewFromFloat(100)

	existing, _ := domain.NewTransfer("key-2", src, dst, amount, "USD")

	idem.On("Get", mock.Anything, "key-2").Return(nil, assert.AnError)
	repo.On("GetByIdempotencyKey", mock.Anything, "key-2").Return(existing, nil)

	tr, err := svc.CreateTransfer(context.Background(), "key-2", src, dst, amount, "USD")
	require.NoError(t, err)
	assert.Equal(t, existing.ID, tr.ID)

	repo.AssertNumberOfCalls(t, "Create", 0)
	outbox.AssertNumberOfCalls(t, "SaveEvent", 0)
}

func TestTransferService_CreateTransfer_InvalidAmount(t *testing.T) {
	repo := new(mockTransferRepo)
	outbox := new(mockOutboxRepo)
	idem := new(mockIdempotencyStore)

	idem.On("Get", mock.Anything, "key-bad").Return(nil, assert.AnError)
	repo.On("GetByIdempotencyKey", mock.Anything, "key-bad").Return(nil, domain.ErrTransferNotFound)

	svc := app.NewTransferService(repo, outbox, idem, zap.NewNop())
	src, dst := uuid.New(), uuid.New()

	_, err := svc.CreateTransfer(context.Background(), "key-bad", src, dst, decimal.Zero, "USD")
	require.Error(t, err)
}

func TestTransferService_HandleTransferCompleted(t *testing.T) {
	repo := new(mockTransferRepo)
	logger := zap.NewNop()
	svc := app.NewTransferService(repo, new(mockOutboxRepo), new(mockIdempotencyStore), logger)

	src, dst := uuid.New(), uuid.New()
	tr, _ := domain.NewTransfer("k", src, dst, decimal.NewFromFloat(50), "USD")

	repo.On("GetByID", mock.Anything, tr.ID).Return(tr, nil)
	repo.On("UpdateWithVersion", mock.Anything, mock.AnythingOfType("*domain.Transfer")).Return(nil)

	err := svc.HandleTransferCompleted(context.Background(), domain.TransferCompletedEvent{
		TransferID: tr.ID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, domain.TransferStatusCompleted, tr.Status)
	repo.AssertExpectations(t)
}

func TestTransferService_HandleTransferFailed(t *testing.T) {
	repo := new(mockTransferRepo)
	logger := zap.NewNop()
	svc := app.NewTransferService(repo, new(mockOutboxRepo), new(mockIdempotencyStore), logger)

	src, dst := uuid.New(), uuid.New()
	tr, _ := domain.NewTransfer("k2", src, dst, decimal.NewFromFloat(75), "USD")

	repo.On("GetByID", mock.Anything, tr.ID).Return(tr, nil)
	repo.On("UpdateWithVersion", mock.Anything, mock.AnythingOfType("*domain.Transfer")).Return(nil)

	err := svc.HandleTransferFailed(context.Background(), domain.TransferFailedEvent{
		TransferID: tr.ID.String(),
		Reason:     "insufficient funds",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.TransferStatusFailed, tr.Status)
	assert.Equal(t, "insufficient funds", tr.FailureReason)
	repo.AssertExpectations(t)
}
