CREATE TABLE IF NOT EXISTS order_timeline (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    order_id UUID NOT NULL
        REFERENCES orders(id)
        ON DELETE CASCADE,

    from_status TEXT,
    to_status TEXT NOT NULL,

    event TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_timeline_from_status
        CHECK (
            from_status IS NULL
            OR from_status IN (
                'PENDING_PAYMENT',
                'QUEUED',
                'IN_PROGRESS',
                'COMPLETED',
                'EXPIRED',
                'FAILED',
                'REFUNDED'
            )
        ),

    CONSTRAINT chk_timeline_to_status
        CHECK (
            to_status IN (
                'PENDING_PAYMENT',
                'QUEUED',
                'IN_PROGRESS',
                'COMPLETED',
                'EXPIRED',
                'FAILED',
                'REFUNDED'
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_order_timeline_order_created
    ON order_timeline (order_id, created_at, id);
