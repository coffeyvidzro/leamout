-- +goose Up
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    email TEXT,
    phone TEXT NOT NULL,
    external_id TEXT,
    address JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_customers_user_id_id ON customers(user_id, id);
CREATE UNIQUE INDEX idx_customers_user_phone ON customers(user_id, phone);
CREATE UNIQUE INDEX idx_customers_user_external_id ON customers(user_id, external_id)
WHERE external_id IS NOT NULL;
CREATE INDEX idx_customers_user_id ON customers(user_id);
CREATE INDEX idx_customers_metadata ON customers USING GIN (metadata);

CREATE TRIGGER customers_set_updated_at
BEFORE UPDATE ON customers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS customers_set_updated_at ON customers;
DROP INDEX IF EXISTS idx_customers_metadata;
DROP INDEX IF EXISTS idx_customers_user_id;
DROP INDEX IF EXISTS idx_customers_user_external_id;
DROP INDEX IF EXISTS idx_customers_user_phone;
DROP INDEX IF EXISTS idx_customers_user_id_id;
DROP TABLE IF EXISTS customers;
