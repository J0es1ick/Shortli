CREATE TABLE IF NOT EXISTS api_key (
    key_id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES user_info(user_id) ON DELETE CASCADE,
    name VARCHAR(60) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    key_hash CHAR(64) UNIQUE NOT NULL,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_api_key_user_active
    ON api_key(user_id, created_at DESC) WHERE revoked_at IS NULL;
