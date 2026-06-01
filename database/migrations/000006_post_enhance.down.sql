ALTER TABLE scheduled_posts
    DROP COLUMN IF EXISTS auto_delete_seconds,
    DROP COLUMN IF EXISTS pin_after_send;
