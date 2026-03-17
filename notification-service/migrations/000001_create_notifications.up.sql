CREATE TABLE IF NOT EXISTS notifications (
    id          UUID        PRIMARY KEY,
    transfer_id VARCHAR(255) NOT NULL,
    channel     VARCHAR(20)  NOT NULL,
    message     TEXT         NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'sent',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_transfer_id ON notifications (transfer_id);
