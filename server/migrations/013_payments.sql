-- +goose Up
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkout_id UUID REFERENCES checkout_sessions(id) ON DELETE SET NULL,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    external_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_reference TEXT,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'authorized', 'captured', 'failed', 'refunded', 'voided')),
    currency TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount >= 0),
    fee_amount BIGINT NOT NULL DEFAULT 0 CHECK (fee_amount >= 0),
    net_amount BIGINT GENERATED ALWAYS AS (amount - fee_amount) STORED,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT payments_user_external_id_key UNIQUE (user_id, external_id)
);

CREATE TABLE IF NOT EXISTS payment_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_reference TEXT,
    status TEXT NOT NULL
        CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'canceled', 'expired', 'unknown')),
    error_code TEXT,
    error_message TEXT,
    raw_request JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_response JSONB NOT NULL DEFAULT '{}'::jsonb,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_attempts_payment_id ON payment_attempts (payment_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_status_created_at ON payments (user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_provider_external_id ON payments (provider, external_id);
CREATE INDEX IF NOT EXISTS idx_payments_provider_reference ON payments (provider, provider_reference) WHERE provider_reference IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payments_checkout_id ON payments (checkout_id);

DROP TRIGGER IF EXISTS payments_set_updated_at ON payments;
CREATE TRIGGER payments_set_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS payments_set_updated_at ON payments;
DROP INDEX IF EXISTS idx_payments_checkout_id;
DROP INDEX IF EXISTS idx_payments_provider_reference;
DROP INDEX IF EXISTS idx_payments_provider_external_id;
DROP INDEX IF EXISTS idx_payments_user_status_created_at;
DROP INDEX IF EXISTS idx_payment_attempts_payment_id;
DROP TABLE IF EXISTS payment_attempts;
DROP TABLE IF EXISTS payments;
