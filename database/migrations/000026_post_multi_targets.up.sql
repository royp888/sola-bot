-- 发帖多渠道投放：scheduled_posts.target_chat_ids_json 保存额外的投放目标
ALTER TABLE scheduled_posts
    ADD COLUMN IF NOT EXISTS target_chat_ids_json TEXT NOT NULL DEFAULT '[]';

-- 投递明细增加状态与错误，便于在面板上看每个频道发成功没有
ALTER TABLE scheduled_post_deliveries
    ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'sent';

ALTER TABLE scheduled_post_deliveries
    ADD COLUMN IF NOT EXISTS error TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_scheduled_post_deliveries_post_chat
    ON scheduled_post_deliveries (scheduled_post_id, chat_id);
