-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_meters_user_id_id
    ON meters (user_id, id);

CREATE TABLE IF NOT EXISTS customer_meters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL,
    meter_id UUID NOT NULL,

    consumed_units NUMERIC NOT NULL DEFAULT 0,
    credited_units NUMERIC NOT NULL DEFAULT 0,
    balance NUMERIC NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_customer_meters_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT fk_customer_meters_meter
        FOREIGN KEY (user_id, meter_id)
        REFERENCES meters (user_id, id)
        ON DELETE CASCADE,

    CONSTRAINT uq_customer_meters_user_customer_meter
        UNIQUE (user_id, customer_id, meter_id),

    CONSTRAINT chk_customer_meters_consumed_units
        CHECK (consumed_units >= 0),

    CONSTRAINT chk_customer_meters_credited_units
        CHECK (credited_units >= 0)
);

CREATE INDEX IF NOT EXISTS idx_customer_meters_user_customer
    ON customer_meters (user_id, customer_id);

CREATE INDEX IF NOT EXISTS idx_customer_meters_user_meter
    ON customer_meters (user_id, meter_id);

CREATE INDEX IF NOT EXISTS idx_customer_meters_user_balance
    ON customer_meters (user_id, balance);

CREATE TRIGGER customer_meters_set_updated_at
BEFORE UPDATE ON customer_meters
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS customer_meters_set_updated_at ON customer_meters;

DROP INDEX IF EXISTS idx_customer_meters_user_balance;
DROP INDEX IF EXISTS idx_customer_meters_user_meter;
DROP INDEX IF EXISTS idx_customer_meters_user_customer;

DROP TABLE IF EXISTS customer_meters;

DROP INDEX IF EXISTS idx_meters_user_id_id;
