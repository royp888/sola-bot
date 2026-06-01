ALTER TABLE lottery_entries
    DROP COLUMN IF EXISTS username;

ALTER TABLE lotteries
    DROP COLUMN IF EXISTS join_keyword,
    DROP COLUMN IF EXISTS join_type;

DROP TABLE IF EXISTS keyword_filters;
DROP TABLE IF EXISTS chat_moderation_configs;
DROP TABLE IF EXISTS violation_records;
DROP TABLE IF EXISTS level_configs;
