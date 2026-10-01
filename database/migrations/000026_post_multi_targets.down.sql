ALTER TABLE scheduled_posts
    DROP COLUMN IF EXISTS target_chat_ids;

ALTER TABLE scheduled_post_deliveries
    DROP COLUMN IF EXISTS status;

ALTER TABLE scheduled_post_deliveries
    DROP COLUMN IF EXISTS error;

DROP INDEX IF EXISTS idx_scheduled_post_deliveries_post_chat;
