package postgres

import (
	"context"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/notification-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationRepository implements app.NotificationRepository backed by PostgreSQL.
type NotificationRepository struct {
	db *pgxpool.Pool
}

// NewNotificationRepository creates a new NotificationRepository.
func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Save persists a notification record.
func (r *NotificationRepository) Save(ctx context.Context, n *domain.Notification) error {
	query := `
		INSERT INTO notifications (id, transfer_id, channel, message, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query, n.ID, n.TransferID, n.Channel, n.Message, n.Status, n.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting notification: %w", err)
	}
	return nil
}
