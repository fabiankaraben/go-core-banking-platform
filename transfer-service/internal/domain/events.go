package domain

// Kafka topic constants mirrored from account-service.
const (
	TopicTransferRequested = "transfers.requested"
	TopicTransferCompleted = "transfers.completed"
	TopicTransferFailed    = "transfers.failed"
)

// TransferRequestedEvent is published by this service to initiate the Saga.
type TransferRequestedEvent struct {
	TransferID      string `json:"transfer_id"`
	SourceAccountID string `json:"source_account_id"`
	DestAccountID   string `json:"dest_account_id"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	TraceID         string `json:"trace_id"`
}

// TransferCompletedEvent is consumed from account-service.
type TransferCompletedEvent struct {
	TransferID string `json:"transfer_id"`
	TraceID    string `json:"trace_id"`
}

// TransferFailedEvent is consumed from account-service.
type TransferFailedEvent struct {
	TransferID string `json:"transfer_id"`
	Reason     string `json:"reason"`
	TraceID    string `json:"trace_id"`
}

// OutboxEvent represents a domain event pending relay to Kafka.
type OutboxEvent struct {
	ID        string
	Topic     string
	Key       string
	Payload   []byte
	Published bool
}
