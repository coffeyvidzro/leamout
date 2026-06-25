-- +goose Up
CREATE TABLE credits (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance BIGINT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'GHS',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_credits_balance CHECK (balance >= 0),
    CONSTRAINT chk_credits_currency CHECK (currency = 'GHS')
);

CREATE TABLE credit_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    provider TEXT,
    destination TEXT,
    reference TEXT,
    description TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_credit_ledger_type CHECK (type IN (
        'topup',
        'debit',
        'refund'
    )),
    CONSTRAINT chk_credit_ledger_amount_direction CHECK (
        (
            type IN ('topup', 'refund')
            AND amount > 0
        )
        OR
        (
            type = 'debit'
            AND amount < 0
        )
    ),
    CONSTRAINT chk_credit_ledger_balance_after CHECK (balance_after >= 0)
);

CREATE INDEX idx_credit_ledger_user_created
ON credit_ledger(user_id, created_at DESC);

CREATE INDEX idx_credit_ledger_reference
ON credit_ledger(reference);

CREATE INDEX idx_credit_ledger_destination
ON credit_ledger(destination);

CREATE TRIGGER credits_set_updated_at
BEFORE UPDATE ON credits
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS credits_set_updated_at ON credits;

DROP INDEX IF EXISTS idx_credit_ledger_destination;
DROP INDEX IF EXISTS idx_credit_ledger_reference;
DROP INDEX IF EXISTS idx_credit_ledger_user_created;

DROP TABLE IF EXISTS credit_ledger;
DROP TABLE IF EXISTS credits;
