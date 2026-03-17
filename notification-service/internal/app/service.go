package app

import (
	"context"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/notification-service/internal/domain"
	"go.uber.org/zap"
)

// NotificationRepository is the outbound port for persisting notifications.
type NotificationRepository interface {
	Save(ctx context.Context, n *domain.Notification) error
}

// NotificationService contains the business logic for processing notification events.
type NotificationService struct {
	repo   NotificationRepository
	logger *zap.Logger
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(repo NotificationRepository, logger *zap.Logger) *NotificationService {
	return &NotificationService{repo: repo, logger: logger}
}

// HandleTransferCompleted processes a successful transfer event.
func (s *NotificationService) HandleTransferCompleted(ctx context.Context, event domain.TransferCompletedEvent) error {
	msg := fmt.Sprintf("Your transfer %s has been completed successfully.", event.TransferID)

	for _, ch := range []string{domain.ChannelEmail, domain.ChannelSMS} {
		n := domain.NewNotification(event.TransferID, ch, msg, domain.StatusSent)
		if err := s.repo.Save(ctx, n); err != nil {
			s.logger.Error("saving completed notification",
				zap.String("transfer_id", event.TransferID),
				zap.String("channel", ch),
				zap.Error(err),
			)
			continue
		}
		s.logger.Info("notification sent",
			zap.String("transfer_id", event.TransferID),
			zap.String("channel", ch),
			zap.String("status", domain.StatusSent),
		)
	}
	return nil
}

// HandleTransferFailed processes a failed transfer event.
func (s *NotificationService) HandleTransferFailed(ctx context.Context, event domain.TransferFailedEvent) error {
	msg := fmt.Sprintf("Your transfer %s has failed: %s.", event.TransferID, event.Reason)

	for _, ch := range []string{domain.ChannelEmail, domain.ChannelSMS} {
		n := domain.NewNotification(event.TransferID, ch, msg, domain.StatusSent)
		if err := s.repo.Save(ctx, n); err != nil {
			s.logger.Error("saving failed notification",
				zap.String("transfer_id", event.TransferID),
				zap.String("channel", ch),
				zap.Error(err),
			)
			continue
		}
		s.logger.Info("failure notification sent",
			zap.String("transfer_id", event.TransferID),
			zap.String("channel", ch),
		)
	}
	return nil
}
