ALTER TABLE url_info
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

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

CREATE INDEX IF NOT EXISTS idx_url_info_expires_at
    ON url_info(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_click_event_url_time
    ON click_event(url_id, clicked_at DESC);
