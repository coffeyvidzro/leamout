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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION record_dunning_attempt_transition()
RETURNS TRIGGER AS $record_dunning_attempt_transition$
DECLARE
    transition_actor TEXT;
    transition_reason TEXT;
    transition_metadata JSONB;
BEGIN
    IF OLD.status IS NOT DISTINCT FROM NEW.status THEN
        RETURN NEW;
    END IF;

    transition_actor := COALESCE(NULLIF(current_setting('leamout.dunning_transition_actor', TRUE), ''), 'system');
    transition_reason := COALESCE(NULLIF(current_setting('leamout.dunning_transition_reason', TRUE), ''), 'status_update');
    transition_metadata := COALESCE(NULLIF(current_setting('leamout.dunning_transition_metadata', TRUE), '')::jsonb, '{}'::jsonb);

    INSERT INTO dunning_attempt_transitions (
        user_id,
        dunning_attempt_id,
        actor,
        reason,
        previous_status,
        next_status,
        metadata
    ) VALUES (
        NEW.user_id,
        NEW.id,
        transition_actor,
        transition_reason,
        OLD.status,
        NEW.status,
        transition_metadata
    );

    RETURN NEW;
END;
$record_dunning_attempt_transition$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER dunning_attempts_record_transition
AFTER UPDATE OF status ON dunning_attempts
FOR EACH ROW EXECUTE FUNCTION record_dunning_attempt_transition();

-- +goose Down
DROP TRIGGER IF EXISTS dunning_attempts_record_transition ON dunning_attempts;
DROP FUNCTION IF EXISTS record_dunning_attempt_transition();
DROP INDEX IF EXISTS idx_dunning_attempt_transitions_metadata;
DROP INDEX IF EXISTS idx_dunning_attempt_transitions_user_created;
DROP INDEX IF EXISTS idx_dunning_attempt_transitions_attempt;
DROP TABLE IF EXISTS dunning_attempt_transitions;
