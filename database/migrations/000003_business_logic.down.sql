DROP INDEX IF EXISTS idx_point_logs_chat_user_created;
DROP INDEX IF EXISTS idx_point_logs_chat_created_user;
DROP INDEX IF EXISTS idx_user_points_chat_total;

ALTER TABLE IF EXISTS lottery_entries
    DROP COLUMN IF EXISTS username;

ALTER TABLE IF EXISTS lotteries
    DROP COLUMN IF EXISTS join_keyword,
    DROP COLUMN IF EXISTS join_type;

DROP TABLE IF EXISTS scheduled_posts;
DROP TABLE IF EXISTS lottery_entries;
DROP TABLE IF EXISTS lotteries;
DROP TABLE IF EXISTS keyword_filters;
DROP TABLE IF EXISTS chat_moderation_configs;
DROP TABLE IF EXISTS violation_records;
DROP TABLE IF EXISTS level_configs;
