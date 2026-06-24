-- +goose Up
CREATE TABLE dunning_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL,
    customer_id UUID,
    status TEXT NOT NULL DEFAULT 'pending',
    reason TEXT NOT NULL DEFAULT 'renewal_due',
    period_end TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_dunning_attempts_user_id_id UNIQUE (user_id, id),
    CONSTRAINT fk_dunning_attempts_user_subscription
        FOREIGN KEY (user_id, subscription_id)
        REFERENCES subscriptions(user_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_dunning_attempts_user_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers(user_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT chk_dunning_attempts_status CHECK (status IN (
        'pending',
        'sent',
        'paid',
        'expired',
        'canceled'
    )),
    CONSTRAINT chk_dunning_attempts_reason CHECK (reason IN (
        'renewal_due',
        'payment_failed'
    )),
    CONSTRAINT chk_dunning_attempts_expiry CHECK (expires_at > created_at),
    CONSTRAINT chk_dunning_attempts_sent_at CHECK (status NOT IN ('sent', 'paid') OR sent_at IS NOT NULL),
    CONSTRAINT chk_dunning_attempts_paid_at CHECK (status <> 'paid' OR paid_at IS NOT NULL),
    CONSTRAINT chk_dunning_attempts_canceled_at CHECK (status <> 'canceled' OR canceled_at IS NOT NULL)
);

CREATE TABLE dunning_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dunning_attempt_id UUID NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_dunning_tokens_user_id_id UNIQUE (user_id, id),
    CONSTRAINT fk_dunning_tokens_user_attempt
        FOREIGN KEY (user_id, dunning_attempt_id)
        REFERENCES dunning_attempts(user_id, id)
        ON DELETE CASCADE,
    CONSTRAINT chk_dunning_tokens_expiry CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX idx_dunning_attempts_active_period
    ON dunning_attempts(user_id, subscription_id, reason, period_end)
    WHERE status IN ('pending', 'sent');
CREATE INDEX idx_dunning_attempts_user_status ON dunning_attempts(user_id, status);
CREATE INDEX idx_dunning_attempts_subscription ON dunning_attempts(user_id, subscription_id, created_at DESC);
CREATE INDEX idx_dunning_attempts_expires_at ON dunning_attempts(expires_at);
CREATE INDEX idx_dunning_attempts_metadata ON dunning_attempts USING GIN (metadata);

CREATE UNIQUE INDEX idx_dunning_tokens_token_hash ON dunning_tokens(token_hash);
CREATE UNIQUE INDEX idx_dunning_tokens_attempt_active
    ON dunning_tokens(dunning_attempt_id)
    WHERE revoked_at IS NULL;
CREATE INDEX idx_dunning_tokens_user_id ON dunning_tokens(user_id);
CREATE INDEX idx_dunning_tokens_expires_at ON dunning_tokens(expires_at);

CREATE TRIGGER dunning_attempts_set_updated_at
BEFORE UPDATE ON dunning_attempts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER dunning_tokens_set_updated_at
BEFORE UPDATE ON dunning_tokens
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS dunning_tokens_set_updated_at ON dunning_tokens;
DROP TRIGGER IF EXISTS dunning_attempts_set_updated_at ON dunning_attempts;
DROP INDEX IF EXISTS idx_dunning_tokens_expires_at;
DROP INDEX IF EXISTS idx_dunning_tokens_user_id;
DROP INDEX IF EXISTS idx_dunning_tokens_attempt_active;
DROP INDEX IF EXISTS idx_dunning_tokens_token_hash;
DROP INDEX IF EXISTS idx_dunning_attempts_metadata;
DROP INDEX IF EXISTS idx_dunning_attempts_expires_at;
DROP INDEX IF EXISTS idx_dunning_attempts_subscription;
DROP INDEX IF EXISTS idx_dunning_attempts_user_status;
DROP INDEX IF EXISTS idx_dunning_attempts_active_period;
DROP TABLE IF EXISTS dunning_tokens;
DROP TABLE IF EXISTS dunning_attempts;
