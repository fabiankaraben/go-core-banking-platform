CREATE TABLE IF NOT EXISTS accounts (
    id          UUID        PRIMARY KEY,
    customer_id UUID        NOT NULL,
    balance     NUMERIC(20, 8) NOT NULL DEFAULT 0,
    currency    VARCHAR(10) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    version     INT         NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounts_customer_id ON accounts (customer_id);
