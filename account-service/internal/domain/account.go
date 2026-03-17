package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Account statuses.
const (
	AccountStatusActive   = "active"
	AccountStatusInactive = "inactive"
	AccountStatusFrozen   = "frozen"
)

// Domain errors.
var (
	ErrAccountNotFound      = errors.New("account not found")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrAccountFrozen        = errors.New("account is frozen")
	ErrCurrencyMismatch     = errors.New("currency mismatch")
	ErrOptimisticLock       = errors.New("optimistic lock conflict: version mismatch")
	ErrInvalidAmount        = errors.New("amount must be positive")
	ErrNegativeBalance      = errors.New("resulting balance cannot be negative")
)

// Account is the core domain aggregate.
type Account struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Balance    decimal.Decimal
	Currency   string
	Status     string
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewAccount creates a new Account with initial balance validation.
func NewAccount(customerID uuid.UUID, currency string, initialBalance decimal.Decimal) (*Account, error) {
	if initialBalance.IsNegative() {
		return nil, ErrNegativeBalance
	}
	return &Account{
		ID:         uuid.New(),
		CustomerID: customerID,
		Balance:    initialBalance,
		Currency:   currency,
		Status:     AccountStatusActive,
		Version:    1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

// Debit subtracts the given amount from the account balance.
// Returns an error if the account is frozen or has insufficient funds.
func (a *Account) Debit(amount decimal.Decimal, currency string) error {
	if amount.IsNegative() || amount.IsZero() {
		return ErrInvalidAmount
	}
	if a.Status == AccountStatusFrozen {
		return ErrAccountFrozen
	}
	if a.Currency != currency {
		return ErrCurrencyMismatch
	}
	if a.Balance.LessThan(amount) {
		return ErrInsufficientFunds
	}
	a.Balance = a.Balance.Sub(amount)
	a.Version++
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// Credit adds the given amount to the account balance.
func (a *Account) Credit(amount decimal.Decimal, currency string) error {
	if amount.IsNegative() || amount.IsZero() {
		return ErrInvalidAmount
	}
	if a.Status == AccountStatusFrozen {
		return ErrAccountFrozen
	}
	if a.Currency != currency {
		return ErrCurrencyMismatch
	}
	a.Balance = a.Balance.Add(amount)
	a.Version++
	a.UpdatedAt = time.Now().UTC()
	return nil
}
