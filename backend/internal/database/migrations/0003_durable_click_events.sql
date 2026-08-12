ALTER TABLE click_event ADD COLUMN IF NOT EXISTS event_key VARCHAR(64);

CREATE UNIQUE INDEX IF NOT EXISTS idx_click_event_event_key
    ON click_event(event_key) WHERE event_key IS NOT NULL;
