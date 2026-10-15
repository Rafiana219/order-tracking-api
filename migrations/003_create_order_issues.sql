CREATE TABLE IF NOT EXISTS order_issues (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    order_id UUID NOT NULL
        REFERENCES orders(id)
        ON DELETE CASCADE,

    description TEXT NOT NULL
        CHECK (LENGTH(TRIM(description)) > 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_issues_order_created
    ON order_issues (order_id, created_at DESC);
