-- +goose Up
CREATE TABLE checkout_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    customer_id UUID,
    subscription_id UUID NOT NULL,
    dunning_attempt_id UUID NOT NULL REFERENCES dunning_attempts(id) ON DELETE CASCADE,
    dunning_token_id UUID NOT NULL REFERENCES dunning_tokens(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'open',
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_checkout_sessions_user_subscription
        FOREIGN KEY (user_id, subscription_id)
        REFERENCES subscriptions(user_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_checkout_sessions_user_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers(user_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT chk_checkout_sessions_status CHECK (status IN (
        'open',
        'completed',
        'expired',
        'canceled'
    )),
    CONSTRAINT chk_checkout_sessions_amount CHECK (amount > 0),
    CONSTRAINT chk_checkout_sessions_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_checkout_sessions_expiry CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX idx_checkout_sessions_user_id_id ON checkout_sessions(user_id, id);
CREATE UNIQUE INDEX idx_checkout_sessions_dunning_token_id ON checkout_sessions(dunning_token_id);
CREATE UNIQUE INDEX idx_checkout_sessions_open_attempt
    ON checkout_sessions(dunning_attempt_id)
    WHERE status = 'open';
CREATE INDEX idx_checkout_sessions_user_status ON checkout_sessions(user_id, status);
CREATE INDEX idx_checkout_sessions_subscription ON checkout_sessions(user_id, subscription_id, created_at DESC);
CREATE INDEX idx_checkout_sessions_expires_at ON checkout_sessions(expires_at);
CREATE INDEX idx_checkout_sessions_metadata ON checkout_sessions USING GIN (metadata);

CREATE TRIGGER checkout_sessions_set_updated_at
BEFORE UPDATE ON checkout_sessions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS checkout_sessions_set_updated_at ON checkout_sessions;
DROP INDEX IF EXISTS idx_checkout_sessions_metadata;
DROP INDEX IF EXISTS idx_checkout_sessions_expires_at;
DROP INDEX IF EXISTS idx_checkout_sessions_subscription;
DROP INDEX IF EXISTS idx_checkout_sessions_user_status;
DROP INDEX IF EXISTS idx_checkout_sessions_open_attempt;
DROP INDEX IF EXISTS idx_checkout_sessions_dunning_token_id;
DROP INDEX IF EXISTS idx_checkout_sessions_user_id_id;
DROP TABLE IF EXISTS checkout_sessions;
