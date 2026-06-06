ALTER TABLE chat_admin_configs
DROP COLUMN IF EXISTS verify_difficulty;

ALTER TABLE chat_admin_configs
DROP COLUMN IF EXISTS verify_whitelist;
