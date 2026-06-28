-- +goose Up
CREATE TABLE IF NOT EXISTS meters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    event_filter JSONB NOT NULL,
    aggregation JSONB NOT NULL,

    unit TEXT NOT NULL DEFAULT 'scalar',
    custom_label TEXT,
    custom_multiplier INTEGER,

    archived_at TIMESTAMPTZ,

    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_meters_name
        CHECK (char_length(trim(name)) >= 3),

    CONSTRAINT chk_meters_unit
        CHECK (unit IN ('scalar', 'token', 'custom')),

    CONSTRAINT chk_meters_custom_unit
        CHECK (
            (
                unit = 'custom'
                AND custom_label IS NOT NULL
                AND char_length(trim(custom_label)) > 0
                AND COALESCE(custom_multiplier, 0) > 0
            )
            OR
            (
                unit <> 'custom'
                AND custom_label IS NULL
                AND custom_multiplier IS NULL
            )
        ),

    CONSTRAINT chk_meters_filter_object
        CHECK (jsonb_typeof(event_filter) = 'object'),

    CONSTRAINT chk_meters_filter_structure
        CHECK (
            event_filter ? 'conjunction'
            AND event_filter ? 'clauses'
            AND event_filter->>'conjunction' IN ('and', 'or')
            AND jsonb_typeof(event_filter->'clauses') = 'array'
        ),

    CONSTRAINT chk_meters_aggregation_object
        CHECK (jsonb_typeof(aggregation) = 'object'),

    CONSTRAINT chk_meters_aggregation_func
        CHECK (
            aggregation ? 'func'
            AND aggregation->>'func' IN ('count', 'sum', 'max', 'min', 'avg', 'unique')
        ),

    CONSTRAINT chk_meters_aggregation_property
        CHECK (
            (
                aggregation->>'func' = 'count'
                AND NOT (aggregation ? 'property')
            )
            OR
            (
                aggregation->>'func' IN ('sum', 'max', 'min', 'avg', 'unique')
                AND aggregation ? 'property'
                AND char_length(trim(aggregation->>'property')) > 0
            )
        ),

    CONSTRAINT chk_meters_metadata_object
        CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_meters_user_name_active
    ON meters (user_id, lower(name))
    WHERE archived_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meters_user_created_at
    ON meters (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_meters_user_active_created_at
    ON meters (user_id, created_at DESC)
    WHERE archived_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_meters_user_archived_at
    ON meters (user_id, archived_at)
    WHERE archived_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_meters_event_filter_gin
    ON meters USING GIN (event_filter);

CREATE INDEX IF NOT EXISTS idx_meters_metadata_gin
    ON meters USING GIN (metadata);

CREATE TRIGGER meters_set_updated_at
BEFORE UPDATE ON meters
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS meters_set_updated_at ON meters;

DROP INDEX IF EXISTS idx_meters_metadata_gin;
DROP INDEX IF EXISTS idx_meters_event_filter_gin;
DROP INDEX IF EXISTS idx_meters_user_archived_at;
DROP INDEX IF EXISTS idx_meters_user_active_created_at;
DROP INDEX IF EXISTS idx_meters_user_created_at;
DROP INDEX IF EXISTS idx_meters_user_name_active;

DROP TABLE IF EXISTS meters;
