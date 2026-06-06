ALTER TABLE chat_admin_configs
ADD COLUMN IF NOT EXISTS verify_whitelist TEXT NOT NULL DEFAULT '';

ALTER TABLE chat_admin_configs
ADD COLUMN IF NOT EXISTS verify_difficulty TEXT NOT NULL DEFAULT 'medium';
