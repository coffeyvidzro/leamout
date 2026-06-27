-- +goose Up
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
    checkout_id UUID REFERENCES checkout_sessions(id) ON DELETE SET NULL,
    external_id TEXT,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    currency TEXT NOT NULL,
    amount BIGINT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_type_occurred_at ON transactions (user_id, type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_payment_id ON transactions (payment_id);

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_payment_id;
DROP INDEX IF EXISTS idx_transactions_user_type_occurred_at;
DROP TABLE IF EXISTS transactions;
