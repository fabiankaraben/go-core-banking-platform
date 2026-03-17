package domain_test

import (
	"testing"

	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransfer(t *testing.T) {
	src := uuid.New()
	dst := uuid.New()

	t.Run("creates transfer with valid inputs", func(t *testing.T) {
		tr, err := domain.NewTransfer("idem-key-1", src, dst, decimal.NewFromFloat(100), "USD")
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusPending, tr.Status)
		assert.Equal(t, "100", tr.Amount.String())
		assert.Equal(t, 1, tr.Version)
	})

	t.Run("rejects zero amount", func(t *testing.T) {
		_, err := domain.NewTransfer("idem-key-2", src, dst, decimal.Zero, "USD")
		assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	})

	t.Run("rejects negative amount", func(t *testing.T) {
		_, err := domain.NewTransfer("idem-key-3", src, dst, decimal.NewFromFloat(-50), "USD")
		assert.ErrorIs(t, err, domain.ErrInvalidAmount)
	})

	t.Run("rejects same source and destination", func(t *testing.T) {
		_, err := domain.NewTransfer("idem-key-4", src, src, decimal.NewFromFloat(100), "USD")
		assert.ErrorIs(t, err, domain.ErrSameAccount)
	})
}

func TestTransfer_MarkCompleted(t *testing.T) {
	src, dst := uuid.New(), uuid.New()
	tr, _ := domain.NewTransfer("key", src, dst, decimal.NewFromFloat(50), "USD")

	tr.MarkCompleted()

	assert.Equal(t, domain.TransferStatusCompleted, tr.Status)
	assert.Equal(t, 2, tr.Version)
}

func TestTransfer_MarkFailed(t *testing.T) {
	src, dst := uuid.New(), uuid.New()
	tr, _ := domain.NewTransfer("key", src, dst, decimal.NewFromFloat(50), "USD")

	tr.MarkFailed("insufficient funds")

	assert.Equal(t, domain.TransferStatusFailed, tr.Status)
	assert.Equal(t, "insufficient funds", tr.FailureReason)
	assert.Equal(t, 2, tr.Version)
}
