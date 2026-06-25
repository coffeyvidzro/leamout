-- +goose Up
CREATE TABLE personal_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_personal_access_tokens_name CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_personal_access_tokens_expiry CHECK (expires_at IS NULL OR expires_at > created_at)
);

CREATE UNIQUE INDEX idx_personal_access_tokens_token_hash
ON personal_access_tokens(token_hash);

CREATE INDEX idx_personal_access_tokens_user_created
ON personal_access_tokens(user_id, created_at DESC);

CREATE INDEX idx_personal_access_tokens_user_active
ON personal_access_tokens(user_id)
WHERE revoked_at IS NULL;

CREATE TRIGGER personal_access_tokens_set_updated_at
BEFORE UPDATE ON personal_access_tokens
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS personal_access_tokens_set_updated_at ON personal_access_tokens;
DROP INDEX IF EXISTS idx_personal_access_tokens_user_active;
DROP INDEX IF EXISTS idx_personal_access_tokens_user_created;
DROP INDEX IF EXISTS idx_personal_access_tokens_token_hash;
DROP TABLE IF EXISTS personal_access_tokens;
