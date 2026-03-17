CREATE TABLE IF NOT EXISTS transfers (
    id                UUID          PRIMARY KEY,
    idempotency_key   VARCHAR(255)  NOT NULL UNIQUE,
    source_account_id UUID          NOT NULL,
    dest_account_id   UUID          NOT NULL,
    amount            NUMERIC(20,8) NOT NULL,
    currency          VARCHAR(10)   NOT NULL,
    status            VARCHAR(20)   NOT NULL DEFAULT 'pending',
    failure_reason    TEXT          NOT NULL DEFAULT '',
    version           INT           NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transfers_idempotency_key ON transfers (idempotency_key);
CREATE INDEX IF NOT EXISTS idx_transfers_status ON transfers (status);
