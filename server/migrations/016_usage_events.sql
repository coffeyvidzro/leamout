-- +goose Up
CREATE TABLE IF NOT EXISTS usage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES usage_events(id) ON DELETE SET NULL,

    name TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'user',

    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    external_customer_id TEXT,
    external_id TEXT,

    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_usage_events_name
        CHECK (char_length(trim(name)) > 0),

    CONSTRAINT chk_usage_events_source
        CHECK (source IN ('system', 'user')),

    CONSTRAINT chk_usage_events_metadata_object
        CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_events_user_external_id
    ON usage_events (user_id, external_id)
    WHERE external_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_usage_events_user_timestamp
    ON usage_events (user_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_usage_events_user_name_timestamp
    ON usage_events (user_id, name, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_usage_events_customer_timestamp
    ON usage_events (customer_id, timestamp DESC)
    WHERE customer_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_usage_events_user_external_customer
    ON usage_events (user_id, external_customer_id)
    WHERE external_customer_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_usage_events_user_source
    ON usage_events (user_id, source);

CREATE INDEX IF NOT EXISTS idx_usage_events_parent_id
    ON usage_events (parent_id)
    WHERE parent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_usage_events_created_at_brin
    ON usage_events USING BRIN (created_at);

CREATE INDEX IF NOT EXISTS idx_usage_events_metadata_gin
    ON usage_events USING GIN (metadata);

-- +goose Down
DROP INDEX IF EXISTS idx_usage_events_metadata_gin;
DROP INDEX IF EXISTS idx_usage_events_created_at_brin;
DROP INDEX IF EXISTS idx_usage_events_parent_id;
DROP INDEX IF EXISTS idx_usage_events_user_source;
DROP INDEX IF EXISTS idx_usage_events_user_external_customer;
DROP INDEX IF EXISTS idx_usage_events_customer_timestamp;
DROP INDEX IF EXISTS idx_usage_events_user_name_timestamp;
DROP INDEX IF EXISTS idx_usage_events_user_timestamp;
DROP INDEX IF EXISTS idx_usage_events_user_external_id;

DROP TABLE IF EXISTS usage_events;
