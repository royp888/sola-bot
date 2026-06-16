ALTER TABLE chat_admin_configs
    ADD COLUMN IF NOT EXISTS rules_text TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS seen_users (
    chat_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chat_id, user_id)
);
