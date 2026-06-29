-- +goose Up
CREATE TABLE dunning_reminder_job_failures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    dunning_attempt_id UUID,
    current_period_end TIMESTAMPTZ NOT NULL,
    failure_number INTEGER NOT NULL,
    status TEXT NOT NULL,
    error_type TEXT NOT NULL,
    error_message TEXT NOT NULL,
    retryable BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_dunning_reminder_job_failures_subscription
        FOREIGN KEY (user_id, subscription_id)
        REFERENCES subscriptions(user_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_dunning_reminder_job_failures_customer
        FOREIGN KEY (user_id, customer_id)
        REFERENCES customers(user_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk_dunning_reminder_job_failures_attempt
        FOREIGN KEY (user_id, dunning_attempt_id)
        REFERENCES dunning_attempts(user_id, id)
        ON DELETE SET NULL,
    CONSTRAINT chk_dunning_reminder_job_failures_failure_number CHECK (failure_number > 0),
    CONSTRAINT chk_dunning_reminder_job_failures_status CHECK (status IN ('retry_scheduled', 'retry_exhausted')),
    CONSTRAINT chk_dunning_reminder_job_failures_error_type CHECK (length(trim(error_type)) > 0),
    CONSTRAINT chk_dunning_reminder_job_failures_error_message CHECK (length(trim(error_message)) > 0)
);

CREATE INDEX idx_dunning_reminder_job_failures_user_created
    ON dunning_reminder_job_failures(user_id, created_at DESC);
CREATE INDEX idx_dunning_reminder_job_failures_attempt
    ON dunning_reminder_job_failures(user_id, dunning_attempt_id, created_at DESC)
    WHERE dunning_attempt_id IS NOT NULL;
CREATE INDEX idx_dunning_reminder_job_failures_status
    ON dunning_reminder_job_failures(user_id, status, created_at DESC);
CREATE INDEX idx_dunning_reminder_job_failures_job_key
    ON dunning_reminder_job_failures(user_id, subscription_id, customer_id, current_period_end, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_dunning_reminder_job_failures_job_key;
DROP INDEX IF EXISTS idx_dunning_reminder_job_failures_status;
DROP INDEX IF EXISTS idx_dunning_reminder_job_failures_attempt;
DROP INDEX IF EXISTS idx_dunning_reminder_job_failures_user_created;
DROP TABLE IF EXISTS dunning_reminder_job_failures;
