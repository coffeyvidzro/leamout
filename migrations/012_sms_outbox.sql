-- +goose Up
CREATE TYPE sms_message_status AS ENUM (
    'pending',
    'debited',
    'sent',
    'failed',
    'refunded'
);

CREATE TABLE sms_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reference TEXT NOT NULL,
    destination TEXT NOT NULL,
    sender TEXT NOT NULL DEFAULT 'Leamout',
    content TEXT NOT NULL,
    country_code TEXT NOT NULL,
    provider TEXT NOT NULL,
    cost BIGINT NOT NULL,
    status sms_message_status NOT NULL DEFAULT 'pending',
    error TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    debited_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    refunded_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_sms_messages_user_reference UNIQUE (user_id, reference),
    CONSTRAINT chk_sms_messages_cost CHECK (cost > 0),
    CONSTRAINT chk_sms_messages_sender CHECK (sender = 'Leamout'),
    CONSTRAINT chk_sms_messages_debited_at CHECK (status NOT IN ('debited', 'sent') OR debited_at IS NOT NULL),
    CONSTRAINT chk_sms_messages_sent_at CHECK (status <> 'sent' OR sent_at IS NOT NULL),
    CONSTRAINT chk_sms_messages_refunded_at CHECK (status <> 'refunded' OR refunded_at IS NOT NULL)
);

CREATE INDEX idx_sms_messages_user_created
ON sms_messages(user_id, created_at DESC);

CREATE INDEX idx_sms_messages_status_created
ON sms_messages(status, created_at);

CREATE UNIQUE INDEX idx_credit_ledger_user_reference_type
ON credit_ledger(user_id, reference, type)
WHERE reference IS NOT NULL;

CREATE TRIGGER sms_messages_set_updated_at
BEFORE UPDATE ON sms_messages
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS sms_messages_set_updated_at ON sms_messages;
DROP INDEX IF EXISTS idx_credit_ledger_user_reference_type;
DROP INDEX IF EXISTS idx_sms_messages_status_created;
DROP INDEX IF EXISTS idx_sms_messages_user_created;
DROP TABLE IF EXISTS sms_messages;
DROP TYPE IF EXISTS sms_message_status;
