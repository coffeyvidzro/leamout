-- +goose Up
CREATE TABLE prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    nickname TEXT NOT NULL,
    type TEXT NOT NULL,
    lookup_key TEXT,
    unit_amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    interval TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_prices_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_prices_type CHECK (type IN ('one_time', 'recurring', 'usage')),
    CONSTRAINT chk_prices_unit_amount CHECK (unit_amount > 0),
    CONSTRAINT chk_prices_interval_value CHECK (interval IS NULL OR interval IN ('day', 'week', 'month', 'year')),
    CONSTRAINT chk_prices_interval_consistency CHECK (
        (type = 'recurring' AND interval IS NOT NULL)
        OR (type <> 'recurring' AND interval IS NULL)
    )
);

CREATE UNIQUE INDEX idx_prices_user_id_id ON prices(user_id, id);
CREATE UNIQUE INDEX idx_prices_user_lookup_key ON prices(user_id, lookup_key)
WHERE lookup_key IS NOT NULL;
CREATE INDEX idx_prices_user_id ON prices(user_id);
CREATE INDEX idx_prices_product_id ON prices(product_id);
CREATE INDEX idx_prices_metadata ON prices USING GIN (metadata);

CREATE TRIGGER prices_set_updated_at
BEFORE UPDATE ON prices
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS prices_set_updated_at ON prices;
DROP INDEX IF EXISTS idx_prices_metadata;
DROP INDEX IF EXISTS idx_prices_product_id;
DROP INDEX IF EXISTS idx_prices_user_id;
DROP INDEX IF EXISTS idx_prices_user_lookup_key;
DROP INDEX IF EXISTS idx_prices_user_id_id;
DROP TABLE IF EXISTS prices;
