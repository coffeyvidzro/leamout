-- +goose Up
CREATE TABLE checkout_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    customer_id UUID,
    subscription_id UUID,
    mode TEXT NOT NULL DEFAULT 'payment',
    source TEXT NOT NULL DEFAULT 'api',
    label TEXT,
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    client_secret_hash TEXT NOT NULL,
    success_url TEXT,
    return_url TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    expires_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_checkout_sessions_user_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers(user_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_checkout_sessions_user_subscription
        FOREIGN KEY (user_id, subscription_id)
        REFERENCES subscriptions(user_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT chk_checkout_sessions_mode CHECK (mode IN (
        'payment',
        'subscription',
        'renewal'
    )),
    CONSTRAINT chk_checkout_sessions_source CHECK (source IN (
        'api',
        'checkout_link',
        'dunning',
        'manual'
    )),
    CONSTRAINT chk_checkout_sessions_status CHECK (status IN (
        'open',
        'completed',
        'expired',
        'canceled'
    )),
    CONSTRAINT chk_checkout_sessions_amount CHECK (amount > 0),
    CONSTRAINT chk_checkout_sessions_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_checkout_sessions_expiry CHECK (expires_at > created_at),
    CONSTRAINT chk_checkout_sessions_completed_at CHECK (status <> 'completed' OR completed_at IS NOT NULL),
    CONSTRAINT chk_checkout_sessions_canceled_at CHECK (status <> 'canceled' OR canceled_at IS NOT NULL)
);

CREATE UNIQUE INDEX idx_checkout_sessions_user_id_id ON checkout_sessions(user_id, id);
CREATE UNIQUE INDEX idx_checkout_sessions_client_secret_hash ON checkout_sessions(client_secret_hash);
CREATE INDEX idx_checkout_sessions_user_status ON checkout_sessions(user_id, status);
CREATE INDEX idx_checkout_sessions_customer ON checkout_sessions(user_id, customer_id, created_at DESC);
CREATE INDEX idx_checkout_sessions_subscription ON checkout_sessions(user_id, subscription_id, created_at DESC);
CREATE INDEX idx_checkout_sessions_source ON checkout_sessions(user_id, source, created_at DESC);
CREATE INDEX idx_checkout_sessions_expires_at ON checkout_sessions(expires_at);
CREATE INDEX idx_checkout_sessions_metadata ON checkout_sessions USING GIN (metadata);

CREATE TRIGGER checkout_sessions_set_updated_at
BEFORE UPDATE ON checkout_sessions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS checkout_sessions_set_updated_at ON checkout_sessions;
DROP INDEX IF EXISTS idx_checkout_sessions_metadata;
DROP INDEX IF EXISTS idx_checkout_sessions_expires_at;
DROP INDEX IF EXISTS idx_checkout_sessions_source;
DROP INDEX IF EXISTS idx_checkout_sessions_subscription;
DROP INDEX IF EXISTS idx_checkout_sessions_customer;
DROP INDEX IF EXISTS idx_checkout_sessions_user_status;
DROP INDEX IF EXISTS idx_checkout_sessions_client_secret_hash;
DROP INDEX IF EXISTS idx_checkout_sessions_user_id_id;
DROP TABLE IF EXISTS checkout_sessions;
