ALTER TABLE chat_moderation_configs
ADD COLUMN IF NOT EXISTS verify_question TEXT NOT NULL DEFAULT '';

ALTER TABLE chat_moderation_configs
ADD COLUMN IF NOT EXISTS verify_options TEXT NOT NULL DEFAULT '[]';

ALTER TABLE chat_moderation_configs
ADD COLUMN IF NOT EXISTS verify_correct_index INTEGER NOT NULL DEFAULT -1;

ALTER TABLE chat_admin_configs
ADD COLUMN IF NOT EXISTS verify_question TEXT NOT NULL DEFAULT '';

ALTER TABLE chat_admin_configs
ADD COLUMN IF NOT EXISTS verify_options TEXT NOT NULL DEFAULT '[]';

ALTER TABLE chat_admin_configs
ADD COLUMN IF NOT EXISTS verify_correct_index INTEGER NOT NULL DEFAULT -1;
