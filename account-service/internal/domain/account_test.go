package domain_test

import (
	"testing"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccount(t *testing.T) {
	customerID := uuid.New()

	t.Run("creates account with valid initial balance", func(t *testing.T) {
		acc, err := domain.NewAccount(customerID, "USD", decimal.NewFromFloat(1000.00))
		require.NoError(t, err)
		assert.Equal(t, "1000", acc.Balance.String())
		assert.Equal(t, "USD", acc.Currency)
		assert.Equal(t, domain.AccountStatusActive, acc.Status)
		assert.Equal(t, 1, acc.Version)
	})

	t.Run("creates account with zero initial balance", func(t *testing.T) {
		acc, err := domain.NewAccount(customerID, "EUR", decimal.Zero)
		require.NoError(t, err)
		assert.True(t, acc.Balance.IsZero())
	})

	t.Run("rejects negative initial balance", func(t *testing.T) {
		_, err := domain.NewAccount(customerID, "USD", decimal.NewFromFloat(-100))
		assert.ErrorIs(t, err, domain.ErrNegativeBalance)
	})
}

func TestAccount_Debit(t *testing.T) {
	t.Run("debit reduces balance and bumps version", func(t *testing.T) {
		acc := newTestAccount("500.00", "USD")
		err := acc.Debit(decimal.NewFromFloat(200), "USD")
		require.NoError(t, err)
		assert.Equal(t, "300", acc.Balance.String())
		assert.Equal(t, 2, acc.Version)
	})

	t.Run("rejects debit exceeding balance", func(t *testing.T) {
		acc := newTestAccount("100.00", "USD")
		err := acc.Debit(decimal.NewFromFloat(200), "USD")
		assert.ErrorIs(t, err, domain.ErrInsufficientFunds)
	})

	t.Run("rejects zero amount", func(t *testing.T) {
		acc := newTestAccount("100.00", "USD")
		err := acc.Debit(decimal.Zero, "USD")
		assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	})

	t.Run("rejects negative amount", func(t *testing.T) {
		acc := newTestAccount("100.00", "USD")
		err := acc.Debit(decimal.NewFromFloat(-50), "USD")
		assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	})

	t.Run("rejects currency mismatch", func(t *testing.T) {
		acc := newTestAccount("500.00", "USD")
		err := acc.Debit(decimal.NewFromFloat(100), "EUR")
		assert.ErrorIs(t, err, domain.ErrCurrencyMismatch)
	})

	t.Run("rejects debit on frozen account", func(t *testing.T) {
		acc := newTestAccount("500.00", "USD")
		acc.Status = domain.AccountStatusFrozen
		err := acc.Debit(decimal.NewFromFloat(100), "USD")
		assert.ErrorIs(t, err, domain.ErrAccountFrozen)
	})
}

func TestAccount_Credit(t *testing.T) {
	t.Run("credit increases balance and bumps version", func(t *testing.T) {
		acc := newTestAccount("100.00", "USD")
		err := acc.Credit(decimal.NewFromFloat(250), "USD")
		require.NoError(t, err)
		assert.Equal(t, "350", acc.Balance.String())
		assert.Equal(t, 2, acc.Version)
	})

	t.Run("rejects zero amount", func(t *testing.T) {
		acc := newTestAccount("100.00", "USD")
		err := acc.Credit(decimal.Zero, "USD")
		assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	})

	t.Run("rejects currency mismatch", func(t *testing.T) {
		acc := newTestAccount("100.00", "USD")
		err := acc.Credit(decimal.NewFromFloat(50), "EUR")
		assert.ErrorIs(t, err, domain.ErrCurrencyMismatch)
	})

	t.Run("rejects credit on frozen account", func(t *testing.T) {
		acc := newTestAccount("100.00", "USD")
		acc.Status = domain.AccountStatusFrozen
		err := acc.Credit(decimal.NewFromFloat(50), "USD")
		assert.ErrorIs(t, err, domain.ErrAccountFrozen)
	})
}

func TestAccount_FinancialPrecision(t *testing.T) {
	t.Run("handles sub-cent precision without floating-point error", func(t *testing.T) {
		acc := newTestAccount("0.1", "USD")
		err := acc.Credit(decimal.RequireFromString("0.2"), "USD")
		require.NoError(t, err)
		assert.Equal(t, "0.3", acc.Balance.String(), "0.1 + 0.2 must equal exactly 0.3")
	})

	t.Run("large monetary values retain precision", func(t *testing.T) {
		acc := newTestAccount("999999999.99999999", "USD")
		err := acc.Debit(decimal.RequireFromString("0.00000001"), "USD")
		require.NoError(t, err)
		assert.Equal(t, "999999999.99999998", acc.Balance.String())
	})
}

// newTestAccount is a helper to create a test Account directly.
func newTestAccount(balance, currency string) *domain.Account {
	acc, _ := domain.NewAccount(uuid.New(), currency, decimal.RequireFromString(balance))
	return acc
}
