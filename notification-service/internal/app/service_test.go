package app_test

import (
	"context"
	"testing"

	"github.com/fabiankaraben/go-core-banking-platform/notification-service/internal/app"
	"github.com/fabiankaraben/go-core-banking-platform/notification-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type mockNotifRepo struct{ mock.Mock }

func (m *mockNotifRepo) Save(ctx context.Context, n *domain.Notification) error {
	return m.Called(ctx, n).Error(0)
}

func TestNotificationService_HandleTransferCompleted(t *testing.T) {
	repo := new(mockNotifRepo)
	logger := zap.NewNop()
	svc := app.NewNotificationService(repo, logger)

	repo.On("Save", mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.TransferID == "txn-123" && n.Status == domain.StatusSent
	})).Return(nil).Times(2)

	err := svc.HandleTransferCompleted(context.Background(), domain.TransferCompletedEvent{
		TransferID: "txn-123",
	})
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestNotificationService_HandleTransferFailed(t *testing.T) {
	repo := new(mockNotifRepo)
	logger := zap.NewNop()
	svc := app.NewNotificationService(repo, logger)

	repo.On("Save", mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.TransferID == "txn-456" && n.Status == domain.StatusSent
	})).Return(nil).Times(2)

	err := svc.HandleTransferFailed(context.Background(), domain.TransferFailedEvent{
		TransferID: "txn-456",
		Reason:     "insufficient funds",
	})
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
