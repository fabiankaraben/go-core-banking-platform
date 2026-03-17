package domain

import (
	"time"

	"github.com/google/uuid"
)

// Notification channel types.
const (
	ChannelEmail = "email"
	ChannelSMS   = "sms"
)

// Notification statuses.
const (
	StatusSent   = "sent"
	StatusFailed = "failed"
)

// Kafka topic constants.
const (
	TopicTransferCompleted = "transfers.completed"
	TopicTransferFailed    = "transfers.failed"
)

// Notification is the core domain entity for the notification-service.
type Notification struct {
	ID         uuid.UUID
	TransferID string
	Channel    string
	Message    string
	Status     string
	CreatedAt  time.Time
}

// NewNotification constructs a new Notification record.
func NewNotification(transferID, channel, message, status string) *Notification {
	return &Notification{
		ID:         uuid.New(),
		TransferID: transferID,
		Channel:    channel,
		Message:    message,
		Status:     status,
		CreatedAt:  time.Now().UTC(),
	}
}

// TransferCompletedEvent is consumed from the account-service.
type TransferCompletedEvent struct {
	TransferID string `json:"transfer_id"`
	TraceID    string `json:"trace_id"`
}

// TransferFailedEvent is consumed from the account-service.
type TransferFailedEvent struct {
	TransferID string `json:"transfer_id"`
	Reason     string `json:"reason"`
	TraceID    string `json:"trace_id"`
}
