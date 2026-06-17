CREATE TABLE IF NOT EXISTS system_settings (
    key        VARCHAR(128) PRIMARY KEY,
    value      TEXT         NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
