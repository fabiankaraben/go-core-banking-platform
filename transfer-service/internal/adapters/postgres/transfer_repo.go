package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// TransferRepository implements ports.TransferRepository backed by PostgreSQL.
type TransferRepository struct {
	db *pgxpool.Pool
}

// NewTransferRepository creates a new TransferRepository.
func NewTransferRepository(db *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{db: db}
}

// Create persists a new transfer record.
func (r *TransferRepository) Create(ctx context.Context, t *domain.Transfer) error {
	query := `
		INSERT INTO transfers (id, idempotency_key, source_account_id, dest_account_id, amount, currency, status, failure_reason, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.Exec(ctx, query,
		t.ID, t.IdempotencyKey, t.SourceAccountID, t.DestAccountID,
		t.Amount.String(), t.Currency, t.Status, t.FailureReason, t.Version, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting transfer: %w", err)
	}
	return nil
}

// GetByID fetches a transfer by primary key.
func (r *TransferRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transfer, error) {
	query := `
		SELECT id, idempotency_key, source_account_id, dest_account_id, amount, currency, status, failure_reason, version, created_at, updated_at
		FROM transfers WHERE id = $1`
	return r.scanTransfer(r.db.QueryRow(ctx, query, id))
}

// GetByIdempotencyKey fetches a transfer by idempotency key.
func (r *TransferRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transfer, error) {
	query := `
		SELECT id, idempotency_key, source_account_id, dest_account_id, amount, currency, status, failure_reason, version, created_at, updated_at
		FROM transfers WHERE idempotency_key = $1`
	return r.scanTransfer(r.db.QueryRow(ctx, query, key))
}

// UpdateWithVersion performs an optimistic-lock update.
func (r *TransferRepository) UpdateWithVersion(ctx context.Context, t *domain.Transfer) error {
	query := `
		UPDATE transfers
		SET status = $1, failure_reason = $2, version = $3, updated_at = $4
		WHERE id = $5 AND version = $6`
	tag, err := r.db.Exec(ctx, query,
		t.Status, t.FailureReason, t.Version, t.UpdatedAt, t.ID, t.Version-1,
	)
	if err != nil {
		return fmt.Errorf("updating transfer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOptimisticLock
	}
	return nil
}

func (r *TransferRepository) scanTransfer(row pgx.Row) (*domain.Transfer, error) {
	var t domain.Transfer
	var amountStr string
	if err := row.Scan(
		&t.ID, &t.IdempotencyKey, &t.SourceAccountID, &t.DestAccountID,
		&amountStr, &t.Currency, &t.Status, &t.FailureReason, &t.Version, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTransferNotFound
		}
		return nil, fmt.Errorf("scanning transfer row: %w", err)
	}
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("parsing amount: %w", err)
	}
	t.Amount = amount
	return &t, nil
}
