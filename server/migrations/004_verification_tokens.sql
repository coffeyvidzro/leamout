-- +goose Up
CREATE TABLE verification_tokens (
    identifier TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (identifier, token_hash)
);

CREATE INDEX idx_verification_tokens_expires_at 
ON verification_tokens(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_verification_tokens_expires_at;
DROP TABLE IF EXISTS verification_tokens;