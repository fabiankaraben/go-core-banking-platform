package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// AccountRepository implements ports.AccountRepository backed by PostgreSQL.
type AccountRepository struct {
	db *pgxpool.Pool
}

// NewAccountRepository creates a new AccountRepository.
func NewAccountRepository(db *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{db: db}
}

// Create persists a new account.
func (r *AccountRepository) Create(ctx context.Context, a *domain.Account) error {
	query := `
		INSERT INTO accounts (id, customer_id, balance, currency, status, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, query,
		a.ID, a.CustomerID, a.Balance.String(), a.Currency, a.Status, a.Version, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting account: %w", err)
	}
	return nil
}

// GetByID fetches an account by its primary key.
func (r *AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	query := `
		SELECT id, customer_id, balance, currency, status, version, created_at, updated_at
		FROM accounts WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)

	var a domain.Account
	var balanceStr string
	if err := row.Scan(&a.ID, &a.CustomerID, &balanceStr, &a.Currency, &a.Status, &a.Version, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("scanning account row: %w", err)
	}
	balance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return nil, fmt.Errorf("parsing balance %q: %w", balanceStr, err)
	}
	a.Balance = balance
	return &a, nil
}

// UpdateWithVersion performs an optimistic-lock update of the account's balance and version.
// It checks the current version matches; if not, returns ErrOptimisticLock.
func (r *AccountRepository) UpdateWithVersion(ctx context.Context, a *domain.Account) error {
	query := `
		UPDATE accounts
		SET balance = $1, version = $2, updated_at = $3
		WHERE id = $4 AND version = $5`
	tag, err := r.db.Exec(ctx, query,
		a.Balance.String(), a.Version, a.UpdatedAt, a.ID, a.Version-1,
	)
	if err != nil {
		return fmt.Errorf("updating account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOptimisticLock
	}
	return nil
}
