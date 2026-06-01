CREATE TABLE IF NOT EXISTS scheduled_post_deliveries (
    id BIGSERIAL PRIMARY KEY,
    scheduled_post_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    auto_delete_at TIMESTAMPTZ,
    auto_deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduled_post_deliveries_post_id
    ON scheduled_post_deliveries (scheduled_post_id);

CREATE INDEX IF NOT EXISTS idx_scheduled_post_deliveries_auto_delete
    ON scheduled_post_deliveries (auto_delete_at)
    WHERE auto_deleted_at IS NULL;
