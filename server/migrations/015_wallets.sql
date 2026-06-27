-- +goose Up
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    country TEXT NOT NULL,
    currency TEXT NOT NULL,
    pending_balance BIGINT NOT NULL DEFAULT 0 CHECK (pending_balance >= 0),
    available_balance BIGINT NOT NULL DEFAULT 0 CHECK (available_balance >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, country, currency)
);

CREATE TABLE IF NOT EXISTS wallet_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
    transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    direction TEXT NOT NULL,
    balance_type TEXT NOT NULL,
    reason TEXT NOT NULL,
    country TEXT NOT NULL,
    currency TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    balance_after BIGINT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wallets_user_country_currency ON wallets (user_id, country, currency);
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_user_created_at ON wallet_ledger_entries (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_country_currency ON wallet_ledger_entries (user_id, country, currency, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_wallet_created_at ON wallet_ledger_entries (wallet_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_ledger_transaction_reason ON wallet_ledger_entries (wallet_id, transaction_id, reason) WHERE transaction_id IS NOT NULL;

CREATE TRIGGER wallets_set_updated_at
BEFORE UPDATE ON wallets
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS wallets_set_updated_at ON wallets;
DROP INDEX IF EXISTS idx_wallet_ledger_transaction_reason;
DROP INDEX IF EXISTS idx_wallet_ledger_wallet_created_at;
DROP INDEX IF EXISTS idx_wallet_ledger_country_currency;
DROP INDEX IF EXISTS idx_wallet_ledger_user_created_at;
DROP INDEX IF EXISTS idx_wallets_user_country_currency;
DROP TABLE IF EXISTS wallet_ledger_entries;
DROP TABLE IF EXISTS wallets;
