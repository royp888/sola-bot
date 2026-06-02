CREATE TABLE IF NOT EXISTS lotteries (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    title VARCHAR(128),
    prize TEXT,
    cost_points INTEGER NOT NULL DEFAULT 0,
    max_participants INTEGER NOT NULL DEFAULT 0,
    winner_count INTEGER NOT NULL DEFAULT 1,
    end_at TIMESTAMPTZ,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    join_type VARCHAR(16) NOT NULL DEFAULT 'button',
    join_keyword VARCHAR(64),
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lottery_entries (
    id BIGSERIAL PRIMARY KEY,
    lottery_id BIGINT NOT NULL REFERENCES lotteries(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    username VARCHAR(64) NOT NULL DEFAULT '',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_winner BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS scheduled_posts (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    title VARCHAR(128),
    content TEXT,
    media_url TEXT,
    media_type VARCHAR(16),
    cron_expr VARCHAR(64),
    run_once_at TIMESTAMPTZ,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE lotteries
    ADD COLUMN IF NOT EXISTS join_type TEXT NOT NULL DEFAULT 'button',
    ADD COLUMN IF NOT EXISTS join_keyword TEXT;

ALTER TABLE lottery_entries
    ADD COLUMN IF NOT EXISTS username TEXT NOT NULL DEFAULT '';

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


CREATE INDEX IF NOT EXISTS idx_user_points_chat_total ON user_points (chat_id, total_points DESC);
CREATE INDEX IF NOT EXISTS idx_point_logs_chat_created_user ON point_logs (chat_id, created_at DESC, user_id);
CREATE INDEX IF NOT EXISTS idx_point_logs_chat_user_created ON point_logs (chat_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_lotteries_chat_id ON lotteries (chat_id);
CREATE INDEX IF NOT EXISTS idx_lotteries_status ON lotteries (status);
CREATE INDEX IF NOT EXISTS idx_lotteries_end_at ON lotteries (end_at);
CREATE INDEX IF NOT EXISTS idx_lotteries_created_by ON lotteries (created_by);
CREATE INDEX IF NOT EXISTS idx_lotteries_chat_status_created ON lotteries (chat_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lotteries_due_active ON lotteries (end_at, id)
    WHERE status = 'active' AND end_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_lotteries_active_keyword_lookup ON lotteries (chat_id, created_at DESC)
    WHERE status = 'active' AND join_type IN ('keyword', 'both') AND join_keyword <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_lottery_entry_user ON lottery_entries (lottery_id, user_id);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_lottery_id ON lottery_entries (lottery_id);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_user_id ON lottery_entries (user_id);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_is_winner ON lottery_entries (is_winner);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_lottery_joined ON lottery_entries (lottery_id, joined_at);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_winners ON lottery_entries (lottery_id, joined_at)
    WHERE is_winner = TRUE;

CREATE INDEX IF NOT EXISTS idx_scheduled_posts_chat_id ON scheduled_posts (chat_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_enabled ON scheduled_posts (enabled);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_run_once_at ON scheduled_posts (run_once_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_chat_created_at ON scheduled_posts (chat_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_chat_enabled ON scheduled_posts (chat_id, enabled);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_due_once ON scheduled_posts (run_once_at, id)
    WHERE enabled = TRUE AND last_run_at IS NULL AND run_once_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_enabled_cron ON scheduled_posts (id)
    WHERE enabled = TRUE AND COALESCE(cron_expr, '') <> '';
