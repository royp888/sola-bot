-- chat_admin_configs was missing the verify_type column that the Go model expects
ALTER TABLE chat_admin_configs
    ADD COLUMN IF NOT EXISTS verify_type TEXT NOT NULL DEFAULT 'button';
