ALTER TABLE chat_moderation_configs
DROP COLUMN IF EXISTS link_blacklist;

ALTER TABLE chat_moderation_configs
DROP COLUMN IF EXISTS link_whitelist;
