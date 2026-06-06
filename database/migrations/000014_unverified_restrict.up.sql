ALTER TABLE chat_moderation_configs
ADD COLUMN IF NOT EXISTS restrict_unverified BOOLEAN NOT NULL DEFAULT TRUE;
