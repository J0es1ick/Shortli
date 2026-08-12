CREATE TABLE IF NOT EXISTS abuse_report (
    report_id BIGSERIAL PRIMARY KEY,
    url_id BIGINT REFERENCES url_info(url_id) ON DELETE SET NULL,
    short_code VARCHAR(100) NOT NULL,
    reporter_email VARCHAR(255),
    reporter_ip_hash CHAR(64) NOT NULL,
    reason VARCHAR(32) NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    resolution_note TEXT NOT NULL DEFAULT '',
    reviewed_by INTEGER REFERENCES user_info(user_id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    CONSTRAINT abuse_report_reason_check CHECK (reason IN ('phishing', 'malware', 'spam', 'impersonation', 'illegal', 'other')),
    CONSTRAINT abuse_report_status_check CHECK (status IN ('pending', 'reviewed', 'dismissed', 'blocked'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_abuse_report_pending_source
    ON abuse_report(url_id, reporter_ip_hash) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_abuse_report_status_created
    ON abuse_report(status, created_at DESC);

CREATE TABLE IF NOT EXISTS blocked_domain (
    domain_id BIGSERIAL PRIMARY KEY,
    domain VARCHAR(253) UNIQUE NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_by INTEGER REFERENCES user_info(user_id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_blocked_domain_domain ON blocked_domain(domain);
