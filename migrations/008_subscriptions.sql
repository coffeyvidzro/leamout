-- +goose Up
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    customer_id UUID,
    price_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ NOT NULL,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    canceled_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    customer_cancellation_reason TEXT,
    customer_cancellation_comment TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_subscriptions_user_price
        FOREIGN KEY (user_id, price_id)
        REFERENCES prices(user_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_subscriptions_user_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers(user_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT chk_subscriptions_status CHECK (status IN (
        'active',
        'canceled',
        'past_due',
        'trialing',
        'incomplete',
        'paused'
    )),
    CONSTRAINT chk_subscriptions_period_bounds CHECK (current_period_end > current_period_start)
);

CREATE UNIQUE INDEX idx_subscriptions_user_id_id ON subscriptions(user_id, id);
CREATE INDEX idx_subscriptions_user_id_created_at ON subscriptions(user_id, created_at DESC);
CREATE INDEX idx_subscriptions_user_customer_created_at ON subscriptions(user_id, customer_id, created_at DESC);
CREATE INDEX idx_subscriptions_user_price_created_at ON subscriptions(user_id, price_id, created_at DESC);
CREATE INDEX idx_subscriptions_user_status ON subscriptions(user_id, status);
CREATE INDEX idx_subscriptions_current_period_end ON subscriptions(current_period_end);
CREATE INDEX idx_subscriptions_renewal_scan ON subscriptions(current_period_end)
WHERE status = 'active'
  AND cancel_at_period_end = FALSE;
CREATE INDEX idx_subscriptions_metadata ON subscriptions USING GIN (metadata);

CREATE TRIGGER subscriptions_set_updated_at
BEFORE UPDATE ON subscriptions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS subscriptions_set_updated_at ON subscriptions;
DROP INDEX IF EXISTS idx_subscriptions_metadata;
DROP INDEX IF EXISTS idx_subscriptions_renewal_scan;
DROP INDEX IF EXISTS idx_subscriptions_current_period_end;
DROP INDEX IF EXISTS idx_subscriptions_user_status;
DROP INDEX IF EXISTS idx_subscriptions_user_price_created_at;
DROP INDEX IF EXISTS idx_subscriptions_user_customer_created_at;
DROP INDEX IF EXISTS idx_subscriptions_user_id_created_at;
DROP INDEX IF EXISTS idx_subscriptions_user_id_id;
DROP TABLE IF EXISTS subscriptions;
