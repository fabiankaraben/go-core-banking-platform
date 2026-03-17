CREATE TABLE IF NOT EXISTS outbox_events (
    id         UUID        PRIMARY KEY,
    topic      VARCHAR(255) NOT NULL,
    key        VARCHAR(255) NOT NULL,
    payload    JSONB       NOT NULL,
    published  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_published ON outbox_events (published, created_at)
    WHERE published = FALSE;
