-- +goose Up
CREATE TABLE IF NOT EXISTS domain_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    idempotency_key TEXT,

    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    last_error TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_domain_events_name CHECK (btrim(name) <> ''),
    CONSTRAINT chk_domain_events_aggregate_type CHECK (btrim(aggregate_type) <> ''),
    CONSTRAINT chk_domain_events_payload_object CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT chk_domain_events_metadata_object CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT chk_domain_events_status CHECK (status IN ('pending', 'processing', 'published', 'failed')),
    CONSTRAINT chk_domain_events_attempts CHECK (attempts >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_domain_events_idempotency_key
    ON domain_events (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_domain_events_ready
    ON domain_events (status, available_at, created_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX IF NOT EXISTS idx_domain_events_user_created
    ON domain_events (user_id, created_at DESC)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_domain_events_aggregate
    ON domain_events (aggregate_type, aggregate_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_domain_events_name_created
    ON domain_events (name, created_at DESC);

CREATE TRIGGER domain_events_set_updated_at
BEFORE UPDATE ON domain_events
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS domain_events_set_updated_at ON domain_events;

DROP INDEX IF EXISTS idx_domain_events_name_created;
DROP INDEX IF EXISTS idx_domain_events_aggregate;
DROP INDEX IF EXISTS idx_domain_events_user_created;
DROP INDEX IF EXISTS idx_domain_events_ready;
DROP INDEX IF EXISTS idx_domain_events_idempotency_key;

DROP TABLE IF EXISTS domain_events;
