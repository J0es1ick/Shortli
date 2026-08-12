CREATE TABLE IF NOT EXISTS user_info (
    user_id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS url_info (
    url_id BIGSERIAL PRIMARY KEY,
    original_url TEXT NOT NULL,
    short_code VARCHAR(100) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES user_info(user_id) ON DELETE SET NULL,
    click_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS click_event (
    event_id BIGSERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES url_info(url_id) ON DELETE CASCADE,
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    device_type VARCHAR(24) NOT NULL DEFAULT 'Unknown',
    browser VARCHAR(40) NOT NULL DEFAULT 'Unknown',
    os VARCHAR(40) NOT NULL DEFAULT 'Unknown',
    referrer_host TEXT NOT NULL DEFAULT 'Direct',
    country_code VARCHAR(8) NOT NULL DEFAULT 'Unknown',
    ip_hash CHAR(64) NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS session_info (
    session_id VARCHAR(64) PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES user_info(user_id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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

CREATE INDEX IF NOT EXISTS idx_url_info_user_id ON url_info(user_id);
CREATE INDEX IF NOT EXISTS idx_url_info_created_at ON url_info(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_url_info_expires_at ON url_info(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_click_event_url_time ON click_event(url_id, clicked_at DESC);
CREATE INDEX IF NOT EXISTS idx_session_info_expires_at ON session_info(expires_at);
CREATE INDEX IF NOT EXISTS idx_api_key_user_active ON api_key(user_id, created_at DESC) WHERE revoked_at IS NULL;
