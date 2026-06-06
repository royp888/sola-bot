ALTER TABLE chat_admin_configs
DROP COLUMN IF EXISTS verify_correct_index;

ALTER TABLE chat_admin_configs
DROP COLUMN IF EXISTS verify_options;

ALTER TABLE chat_admin_configs
DROP COLUMN IF EXISTS verify_question;

ALTER TABLE chat_moderation_configs
DROP COLUMN IF EXISTS verify_correct_index;

ALTER TABLE chat_moderation_configs
DROP COLUMN IF EXISTS verify_options;

ALTER TABLE chat_moderation_configs
DROP COLUMN IF EXISTS verify_question;
