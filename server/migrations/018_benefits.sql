-- +goose Up
CREATE TABLE IF NOT EXISTS benefits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    type TEXT NOT NULL,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT,

    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    archived_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_benefits_type
        CHECK (type IN ('custom', 'feature', 'meter_credit')),

    CONSTRAINT chk_benefits_name
        CHECK (char_length(trim(name)) >= 3),

    CONSTRAINT chk_benefits_code
        CHECK (char_length(trim(code)) >= 2),

    CONSTRAINT chk_benefits_properties_object
        CHECK (jsonb_typeof(properties) = 'object'),

    CONSTRAINT chk_benefits_metadata_object
        CHECK (jsonb_typeof(metadata) = 'object'),

    CONSTRAINT chk_benefits_meter_credit_properties
        CHECK (
            type <> 'meter_credit'
            OR (
                properties ? 'meter_id'
                AND properties ? 'quantity'
                AND char_length(trim(properties->>'meter_id')) > 0
                AND (properties->>'quantity') ~ '^[0-9]+$'
                AND (properties->>'quantity')::BIGINT > 0
            )
        )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_benefits_user_id_id
    ON benefits (user_id, id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_benefits_user_code_active
    ON benefits (user_id, lower(code))
    WHERE archived_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_benefits_user_created_at
    ON benefits (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_benefits_user_type
    ON benefits (user_id, type);

CREATE INDEX IF NOT EXISTS idx_benefits_user_active_created_at
    ON benefits (user_id, created_at DESC)
    WHERE archived_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_benefits_properties_gin
    ON benefits USING GIN (properties);

CREATE INDEX IF NOT EXISTS idx_benefits_metadata_gin
    ON benefits USING GIN (metadata);

CREATE TABLE IF NOT EXISTS product_benefits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    benefit_id UUID NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_product_benefits_product
        FOREIGN KEY (user_id, product_id)
        REFERENCES products (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT fk_product_benefits_benefit
        FOREIGN KEY (user_id, benefit_id)
        REFERENCES benefits (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT uq_product_benefits_user_product_benefit
        UNIQUE (user_id, product_id, benefit_id)
);

CREATE INDEX IF NOT EXISTS idx_product_benefits_user_product
    ON product_benefits (user_id, product_id);

CREATE INDEX IF NOT EXISTS idx_product_benefits_user_benefit
    ON product_benefits (user_id, benefit_id);

CREATE TABLE IF NOT EXISTS benefit_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    benefit_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    product_id UUID,
    subscription_id UUID,

    source_type TEXT NOT NULL,
    source_id UUID NOT NULL,

    status TEXT NOT NULL DEFAULT 'active',

    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,

    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,

    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_benefit_grants_benefit
        FOREIGN KEY (user_id, benefit_id)
        REFERENCES benefits (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT fk_benefit_grants_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT fk_benefit_grants_product
        FOREIGN KEY (user_id, product_id)
        REFERENCES products (user_id, id)
        ON DELETE SET NULL,

    CONSTRAINT fk_benefit_grants_subscription
        FOREIGN KEY (user_id, subscription_id)
        REFERENCES subscriptions (user_id, id)
        ON DELETE SET NULL,

    CONSTRAINT chk_benefit_grants_source_type
        CHECK (source_type IN ('subscription', 'manual')),

    CONSTRAINT chk_benefit_grants_status
        CHECK (status IN ('active', 'revoked', 'expired')),

    CONSTRAINT chk_benefit_grants_source_link
        CHECK (
            (source_type = 'subscription' AND subscription_id IS NOT NULL)
            OR
            (source_type = 'manual')
        ),

    CONSTRAINT chk_benefit_grants_period
        CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at),

    CONSTRAINT chk_benefit_grants_timeline
        CHECK (revoked_at IS NULL OR revoked_at >= granted_at),

    CONSTRAINT chk_benefit_grants_properties_object
        CHECK (jsonb_typeof(properties) = 'object'),

    CONSTRAINT chk_benefit_grants_metadata_object
        CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_benefit_grants_user_scope
    ON benefit_grants (user_id, customer_id, benefit_id, source_type, source_id);

CREATE INDEX IF NOT EXISTS idx_benefit_grants_user_customer_active
    ON benefit_grants (user_id, customer_id, created_at DESC)
    WHERE status = 'active' AND revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_benefit_grants_user_benefit_created_at
    ON benefit_grants (user_id, benefit_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_benefit_grants_user_subscription
    ON benefit_grants (user_id, subscription_id)
    WHERE subscription_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_benefit_grants_properties_gin
    ON benefit_grants USING GIN (properties);

CREATE INDEX IF NOT EXISTS idx_benefit_grants_metadata_gin
    ON benefit_grants USING GIN (metadata);

CREATE TRIGGER benefits_set_updated_at
BEFORE UPDATE ON benefits
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER benefit_grants_set_updated_at
BEFORE UPDATE ON benefit_grants
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS benefit_grants_set_updated_at ON benefit_grants;
DROP TRIGGER IF EXISTS benefits_set_updated_at ON benefits;

DROP INDEX IF EXISTS idx_benefit_grants_metadata_gin;
DROP INDEX IF EXISTS idx_benefit_grants_properties_gin;
DROP INDEX IF EXISTS idx_benefit_grants_user_subscription;
DROP INDEX IF EXISTS idx_benefit_grants_user_benefit_created_at;
DROP INDEX IF EXISTS idx_benefit_grants_user_customer_active;
DROP INDEX IF EXISTS idx_benefit_grants_user_scope;

DROP INDEX IF EXISTS idx_product_benefits_user_benefit;
DROP INDEX IF EXISTS idx_product_benefits_user_product;

DROP INDEX IF EXISTS idx_benefits_metadata_gin;
DROP INDEX IF EXISTS idx_benefits_properties_gin;
DROP INDEX IF EXISTS idx_benefits_user_active_created_at;
DROP INDEX IF EXISTS idx_benefits_user_type;
DROP INDEX IF EXISTS idx_benefits_user_created_at;
DROP INDEX IF EXISTS idx_benefits_user_code_active;
DROP INDEX IF EXISTS idx_benefits_user_id_id;

DROP TABLE IF EXISTS benefit_grants;
DROP TABLE IF EXISTS product_benefits;
DROP TABLE IF EXISTS benefits;
