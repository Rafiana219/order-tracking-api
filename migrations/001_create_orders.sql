CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    customer_id TEXT NOT NULL,
    items JSONB NOT NULL,

    status TEXT NOT NULL DEFAULT 'PENDING_PAYMENT'
        CHECK (
            status IN (
                'PENDING_PAYMENT',
                'QUEUED',
                'IN_PROGRESS',
                'COMPLETED',
                'EXPIRED',
                'FAILED',
                'REFUNDED'
            )
        ),

    idempotency_key TEXT NOT NULL UNIQUE,
    request_hash TEXT NOT NULL,

    attempts INTEGER NOT NULL DEFAULT 0
        CHECK (attempts >= 0),

    max_attempts INTEGER NOT NULL DEFAULT 3
        CHECK (max_attempts >= 1),

    next_attempt_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_status_created
    ON orders (status, created_at);

CREATE INDEX IF NOT EXISTS idx_orders_retry
    ON orders (status, next_attempt_at);
