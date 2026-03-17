package postgres

import (
	"context"
	"fmt"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxRepository implements ports.OutboxRepository backed by PostgreSQL.
type OutboxRepository struct {
	db *pgxpool.Pool
}

// NewOutboxRepository creates a new OutboxRepository.
func NewOutboxRepository(db *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// SaveEvent persists a new outbox event.
func (r *OutboxRepository) SaveEvent(ctx context.Context, event *domain.OutboxEvent) error {
	query := `
		INSERT INTO outbox_events (id, topic, key, payload, published)
		VALUES ($1, $2, $3, $4, false)`
	_, err := r.db.Exec(ctx, query, event.ID, event.Topic, event.Key, event.Payload)
	if err != nil {
		return fmt.Errorf("inserting outbox event: %w", err)
	}
	return nil
}

// GetUnpublished retrieves unpublished outbox events, up to the given limit.
func (r *OutboxRepository) GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	query := `
		SELECT id, topic, key, payload
		FROM outbox_events
		WHERE published = false
		ORDER BY created_at ASC
		LIMIT $1`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("querying outbox events: %w", err)
	}
	defer rows.Close()

	var events []*domain.OutboxEvent
	for rows.Next() {
		var e domain.OutboxEvent
		if err := rows.Scan(&e.ID, &e.Topic, &e.Key, &e.Payload); err != nil {
			return nil, fmt.Errorf("scanning outbox event: %w", err)
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

// MarkPublished marks an outbox event as published.
func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	query := `UPDATE outbox_events SET published = true WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("marking outbox event published: %w", err)
	}
	return nil
}
