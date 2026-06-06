ALTER TABLE chat_moderation_configs
ADD COLUMN IF NOT EXISTS link_whitelist TEXT NOT NULL DEFAULT '';

ALTER TABLE chat_moderation_configs
ADD COLUMN IF NOT EXISTS link_blacklist TEXT NOT NULL DEFAULT '';
