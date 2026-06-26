-- +goose Up
CREATE TABLE checkout_payment_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checkout_session_id UUID NOT NULL REFERENCES checkout_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    external_ref TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    provider_reference TEXT,
    status TEXT NOT NULL DEFAULT 'pending',

    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    country TEXT NOT NULL,
    payment_method TEXT NOT NULL,
    operator TEXT,

    customer_phone TEXT,
    provider_response JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_checkout_payment_attempts_external_ref UNIQUE (external_ref),
    CONSTRAINT chk_checkout_payment_attempts_status CHECK (status IN (
        'pending',
        'processing',
        'succeeded',
        'failed',
        'canceled',
        'expired',
        'unknown'
    )),
    CONSTRAINT chk_checkout_payment_attempts_amount CHECK (amount > 0),
    CONSTRAINT chk_checkout_payment_attempts_currency CHECK (currency ~ '^[A-Z]{3}$')
);

CREATE INDEX idx_checkout_payment_attempts_session ON checkout_payment_attempts(checkout_session_id, created_at DESC);
CREATE INDEX idx_checkout_payment_attempts_provider_ref ON checkout_payment_attempts(provider_id, provider_reference);
CREATE INDEX idx_checkout_payment_attempts_status ON checkout_payment_attempts(status);
CREATE INDEX idx_checkout_payment_attempts_metadata ON checkout_payment_attempts USING GIN (metadata);

CREATE TRIGGER checkout_payment_attempts_set_updated_at
BEFORE UPDATE ON checkout_payment_attempts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS checkout_payment_attempts_set_updated_at ON checkout_payment_attempts;
DROP INDEX IF EXISTS idx_checkout_payment_attempts_metadata;
DROP INDEX IF EXISTS idx_checkout_payment_attempts_status;
DROP INDEX IF EXISTS idx_checkout_payment_attempts_provider_ref;
DROP INDEX IF EXISTS idx_checkout_payment_attempts_session;
DROP TABLE IF EXISTS checkout_payment_attempts;
