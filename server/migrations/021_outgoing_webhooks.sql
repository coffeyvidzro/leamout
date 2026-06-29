-- +goose Up
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    description TEXT,
    secret_hash TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    event_types TEXT[] NOT NULL DEFAULT ARRAY['*']::TEXT[],
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_webhook_endpoints_url CHECK (url ~* '^https?://'),
    CONSTRAINT chk_webhook_endpoints_secret_hash CHECK (btrim(secret_hash) <> ''),
    CONSTRAINT chk_webhook_endpoints_event_types CHECK (array_length(event_types, 1) > 0),
    CONSTRAINT chk_webhook_endpoints_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_user_created
    ON webhook_endpoints (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_user_enabled
    ON webhook_endpoints (user_id, enabled)
    WHERE enabled = TRUE;

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    idempotency_key TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,

    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    last_status_code INTEGER,
    last_error TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_webhook_deliveries_event_type CHECK (btrim(event_type) <> ''),
    CONSTRAINT chk_webhook_deliveries_aggregate_type CHECK (btrim(aggregate_type) <> ''),
    CONSTRAINT chk_webhook_deliveries_payload_object CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT chk_webhook_deliveries_status CHECK (status IN ('pending', 'processing', 'delivered', 'failed')),
    CONSTRAINT chk_webhook_deliveries_attempts CHECK (attempts >= 0),
    CONSTRAINT chk_webhook_deliveries_status_code CHECK (last_status_code IS NULL OR last_status_code > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint_idempotency_key
    ON webhook_deliveries (endpoint_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_ready
    ON webhook_deliveries (status, next_attempt_at, created_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint_created
    ON webhook_deliveries (endpoint_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_user_created
    ON webhook_deliveries (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event
    ON webhook_deliveries (event_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_aggregate
    ON webhook_deliveries (aggregate_type, aggregate_id, created_at DESC);

CREATE TRIGGER webhook_endpoints_set_updated_at
BEFORE UPDATE ON webhook_endpoints
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER webhook_deliveries_set_updated_at
BEFORE UPDATE ON webhook_deliveries
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS webhook_deliveries_set_updated_at ON webhook_deliveries;
DROP TRIGGER IF EXISTS webhook_endpoints_set_updated_at ON webhook_endpoints;

DROP INDEX IF EXISTS idx_webhook_deliveries_aggregate;
DROP INDEX IF EXISTS idx_webhook_deliveries_event;
DROP INDEX IF EXISTS idx_webhook_deliveries_user_created;
DROP INDEX IF EXISTS idx_webhook_deliveries_endpoint_created;
DROP INDEX IF EXISTS idx_webhook_deliveries_ready;
DROP INDEX IF EXISTS idx_webhook_deliveries_endpoint_idempotency_key;

DROP TABLE IF EXISTS webhook_deliveries;

DROP INDEX IF EXISTS idx_webhook_endpoints_user_enabled;
DROP INDEX IF EXISTS idx_webhook_endpoints_user_created;

DROP TABLE IF EXISTS webhook_endpoints;
