-- +goose Up
CREATE TABLE dunning_attempt_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    dunning_attempt_id UUID NOT NULL,
    actor TEXT NOT NULL,
    reason TEXT NOT NULL,
    previous_status TEXT NOT NULL,
    next_status TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_dunning_attempt_transitions_user_attempt
        FOREIGN KEY (user_id, dunning_attempt_id)
        REFERENCES dunning_attempts(user_id, id)
        ON DELETE CASCADE,
    CONSTRAINT chk_dunning_attempt_transitions_actor CHECK (length(trim(actor)) > 0),
    CONSTRAINT chk_dunning_attempt_transitions_reason CHECK (length(trim(reason)) > 0),
    CONSTRAINT chk_dunning_attempt_transitions_previous_status CHECK (previous_status IN (
        'pending',
        'sent',
        'paid',
        'expired',
        'canceled'
    )),
    CONSTRAINT chk_dunning_attempt_transitions_next_status CHECK (next_status IN (
        'pending',
        'sent',
        'paid',
        'expired',
        'canceled'
    )),
    CONSTRAINT chk_dunning_attempt_transitions_status_change CHECK (previous_status <> next_status)
);

CREATE INDEX idx_dunning_attempt_transitions_attempt
    ON dunning_attempt_transitions(user_id, dunning_attempt_id, created_at DESC);
CREATE INDEX idx_dunning_attempt_transitions_user_created
    ON dunning_attempt_transitions(user_id, created_at DESC);
CREATE INDEX idx_dunning_attempt_transitions_metadata
    ON dunning_attempt_transitions USING GIN (metadata);

-- +goose Down
DROP INDEX IF EXISTS idx_dunning_attempt_transitions_metadata;
DROP INDEX IF EXISTS idx_dunning_attempt_transitions_user_created;
DROP INDEX IF EXISTS idx_dunning_attempt_transitions_attempt;
DROP TABLE IF EXISTS dunning_attempt_transitions;
