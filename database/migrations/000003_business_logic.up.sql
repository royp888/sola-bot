CREATE TABLE IF NOT EXISTS level_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id BIGINT NOT NULL,
    level INTEGER NOT NULL,
    min_points BIGINT NOT NULL DEFAULT 0,
    label TEXT NOT NULL DEFAULT '',
    can_post_link BOOLEAN NOT NULL DEFAULT TRUE,
    can_post_media BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT idx_level_configs_chat_level UNIQUE (chat_id, level)
);

CREATE TABLE IF NOT EXISTS violation_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    violation_type TEXT NOT NULL,
    action_taken TEXT NOT NULL,
    message_text TEXT,
    detected_by TEXT NOT NULL DEFAULT 'rule',
    duration_seconds INTEGER DEFAULT 0,
    cleared BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS chat_moderation_configs (
    chat_id BIGINT PRIMARY KEY,
    verify_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    verify_type TEXT NOT NULL DEFAULT 'button',
    verify_timeout_seconds INTEGER NOT NULL DEFAULT 60,
    warn_limit INTEGER NOT NULL DEFAULT 3,
    block_links BOOLEAN NOT NULL DEFAULT FALSE,
    block_forwards BOOLEAN NOT NULL DEFAULT FALSE,
    block_media BOOLEAN NOT NULL DEFAULT FALSE,
    keyword_filter_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    spam_score_threshold INTEGER NOT NULL DEFAULT 60,
    welcome_text TEXT NOT NULL DEFAULT '欢迎 {name}！',
    welcome_delete_seconds INTEGER NOT NULL DEFAULT 30,
    updated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS keyword_filters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id BIGINT NOT NULL,
    keyword TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'contains',
    action TEXT NOT NULL DEFAULT 'delete',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT idx_keyword_filters_chat_keyword UNIQUE (chat_id, keyword)
);

CREATE INDEX IF NOT EXISTS idx_level_configs_chat_id ON level_configs (chat_id);
CREATE INDEX IF NOT EXISTS idx_violation_records_user_chat ON violation_records (user_id, chat_id);
CREATE INDEX IF NOT EXISTS idx_violation_records_chat_id ON violation_records (chat_id);
CREATE INDEX IF NOT EXISTS idx_violation_records_violation_type ON violation_records (violation_type);
CREATE INDEX IF NOT EXISTS idx_violation_records_action_taken ON violation_records (action_taken);
CREATE INDEX IF NOT EXISTS idx_violation_records_detected_by ON violation_records (detected_by);
CREATE INDEX IF NOT EXISTS idx_violation_records_cleared ON violation_records (cleared);
CREATE INDEX IF NOT EXISTS idx_keyword_filters_chat_id ON keyword_filters (chat_id);
CREATE INDEX IF NOT EXISTS idx_keyword_filters_created_by ON keyword_filters (created_by);

ALTER TABLE lotteries
    ADD COLUMN IF NOT EXISTS join_type TEXT NOT NULL DEFAULT 'button',
    ADD COLUMN IF NOT EXISTS join_keyword TEXT;

ALTER TABLE lottery_entries
    ADD COLUMN IF NOT EXISTS username TEXT NOT NULL DEFAULT '';
