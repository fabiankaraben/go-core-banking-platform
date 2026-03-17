package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Transfer statuses.
const (
	TransferStatusPending   = "pending"
	TransferStatusCompleted = "completed"
	TransferStatusFailed    = "failed"
)

// Domain errors.
var (
	ErrTransferNotFound    = errors.New("transfer not found")
	ErrDuplicateRequest    = errors.New("duplicate idempotency key")
	ErrInvalidAmount       = errors.New("amount must be positive")
	ErrSameAccount         = errors.New("source and destination accounts must differ")
	ErrOptimisticLock      = errors.New("optimistic lock conflict: version mismatch")
)

// Transfer is the core domain aggregate for the transfer-service.
type Transfer struct {
	ID              uuid.UUID
	IdempotencyKey  string
	SourceAccountID uuid.UUID
	DestAccountID   uuid.UUID
	Amount          decimal.Decimal
	Currency        string
	Status          string
	FailureReason   string
	Version         int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewTransfer constructs a Transfer, validating business invariants.
func NewTransfer(idempotencyKey string, sourceID, destID uuid.UUID, amount decimal.Decimal, currency string) (*Transfer, error) {
	if amount.IsNegative() || amount.IsZero() {
		return nil, ErrInvalidAmount
	}
	if sourceID == destID {
		return nil, ErrSameAccount
	}
	return &Transfer{
		ID:              uuid.New(),
		IdempotencyKey:  idempotencyKey,
		SourceAccountID: sourceID,
		DestAccountID:   destID,
		Amount:          amount,
		Currency:        currency,
		Status:          TransferStatusPending,
		Version:         1,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}, nil
}

// MarkCompleted transitions the transfer to the completed state.
func (t *Transfer) MarkCompleted() {
	t.Status = TransferStatusCompleted
	t.Version++
	t.UpdatedAt = time.Now().UTC()
}

// MarkFailed transitions the transfer to the failed state with a reason.
func (t *Transfer) MarkFailed(reason string) {
	t.Status = TransferStatusFailed
	t.FailureReason = reason
	t.Version++
	t.UpdatedAt = time.Now().UTC()
}
