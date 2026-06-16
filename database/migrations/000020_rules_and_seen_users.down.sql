ALTER TABLE chat_admin_configs
    DROP COLUMN IF EXISTS rules_text;

DROP TABLE IF EXISTS seen_users;
