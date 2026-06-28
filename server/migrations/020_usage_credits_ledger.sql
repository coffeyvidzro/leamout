-- +goose Up
CREATE TABLE IF NOT EXISTS meter_credit_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL,
    meter_id UUID NOT NULL,
    benefit_grant_id UUID NOT NULL REFERENCES benefit_grants(id) ON DELETE CASCADE,
    subscription_id UUID,

    source_type TEXT NOT NULL,
    source_id UUID NOT NULL,

    status TEXT NOT NULL DEFAULT 'active',
    quantity NUMERIC NOT NULL,
    remaining_quantity NUMERIC NOT NULL,

    starts_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    rollover_enabled BOOLEAN NOT NULL DEFAULT FALSE,

    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_meter_credit_grants_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT fk_meter_credit_grants_meter
        FOREIGN KEY (user_id, meter_id)
        REFERENCES meters (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT fk_meter_credit_grants_subscription
        FOREIGN KEY (user_id, subscription_id)
        REFERENCES subscriptions (user_id, id)
        ON DELETE SET NULL,

    CONSTRAINT chk_meter_credit_grants_source_type
        CHECK (source_type IN ('checkout', 'manual', 'system')),

    CONSTRAINT chk_meter_credit_grants_status
        CHECK (status IN ('active', 'depleted', 'expired', 'voided')),

    CONSTRAINT chk_meter_credit_grants_quantity
        CHECK (quantity > 0),

    CONSTRAINT chk_meter_credit_grants_remaining_quantity
        CHECK (remaining_quantity >= 0 AND remaining_quantity <= quantity),

    CONSTRAINT chk_meter_credit_grants_period
        CHECK (expires_at IS NULL OR starts_at IS NULL OR expires_at > starts_at),

    CONSTRAINT chk_meter_credit_grants_metadata_object
        CHECK (jsonb_typeof(metadata) = 'object'),

    CONSTRAINT uq_meter_credit_grants_source
        UNIQUE (user_id, source_type, source_id, benefit_grant_id, meter_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_meter_credit_grants_user_id_id
    ON meter_credit_grants (user_id, id);

CREATE INDEX IF NOT EXISTS idx_meter_credit_grants_user_customer
    ON meter_credit_grants (user_id, customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_meter_credit_grants_user_meter
    ON meter_credit_grants (user_id, meter_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_meter_credit_grants_user_subscription
    ON meter_credit_grants (user_id, subscription_id, created_at DESC)
    WHERE subscription_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_meter_credit_grants_user_active
    ON meter_credit_grants (user_id, customer_id, meter_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_meter_credit_grants_metadata_gin
    ON meter_credit_grants USING GIN (metadata);

CREATE TABLE IF NOT EXISTS meter_credit_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    grant_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    meter_id UUID NOT NULL,
    usage_event_id UUID REFERENCES usage_events(id) ON DELETE SET NULL,

    direction TEXT NOT NULL,
    reason TEXT NOT NULL,
    quantity NUMERIC NOT NULL,
    balance_after NUMERIC NOT NULL,

    idempotency_key TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_meter_credit_ledger_grant
        FOREIGN KEY (user_id, grant_id)
        REFERENCES meter_credit_grants (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT fk_meter_credit_ledger_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT fk_meter_credit_ledger_meter
        FOREIGN KEY (user_id, meter_id)
        REFERENCES meters (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT chk_meter_credit_ledger_direction
        CHECK (direction IN ('credit', 'debit')),

    CONSTRAINT chk_meter_credit_ledger_reason
        CHECK (reason IN ('grant', 'consume', 'expire', 'adjust', 'refund')),

    CONSTRAINT chk_meter_credit_ledger_quantity
        CHECK (quantity > 0),

    CONSTRAINT chk_meter_credit_ledger_metadata_object
        CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_meter_credit_ledger_user_idempotency_key
    ON meter_credit_ledger_entries (user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_meter_credit_ledger_user_customer
    ON meter_credit_ledger_entries (user_id, customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_meter_credit_ledger_user_meter
    ON meter_credit_ledger_entries (user_id, meter_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_meter_credit_ledger_user_grant
    ON meter_credit_ledger_entries (user_id, grant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_meter_credit_ledger_usage_event
    ON meter_credit_ledger_entries (usage_event_id)
    WHERE usage_event_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_meter_credit_ledger_metadata_gin
    ON meter_credit_ledger_entries USING GIN (metadata);

CREATE TRIGGER meter_credit_grants_set_updated_at
BEFORE UPDATE ON meter_credit_grants
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS meter_credit_grants_set_updated_at ON meter_credit_grants;

DROP INDEX IF EXISTS idx_meter_credit_ledger_metadata_gin;
DROP INDEX IF EXISTS idx_meter_credit_ledger_usage_event;
DROP INDEX IF EXISTS idx_meter_credit_ledger_user_grant;
DROP INDEX IF EXISTS idx_meter_credit_ledger_user_meter;
DROP INDEX IF EXISTS idx_meter_credit_ledger_user_customer;
DROP INDEX IF EXISTS idx_meter_credit_ledger_user_idempotency_key;

DROP TABLE IF EXISTS meter_credit_ledger_entries;

DROP INDEX IF EXISTS idx_meter_credit_grants_metadata_gin;
DROP INDEX IF EXISTS idx_meter_credit_grants_user_active;
DROP INDEX IF EXISTS idx_meter_credit_grants_user_subscription;
DROP INDEX IF EXISTS idx_meter_credit_grants_user_meter;
DROP INDEX IF EXISTS idx_meter_credit_grants_user_customer;
DROP INDEX IF EXISTS idx_meter_credit_grants_user_id_id;

DROP TABLE IF EXISTS meter_credit_grants;
