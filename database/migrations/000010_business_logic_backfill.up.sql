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

ALTER TABLE lotteries
    ADD COLUMN IF NOT EXISTS join_type TEXT NOT NULL DEFAULT 'button',
    ADD COLUMN IF NOT EXISTS join_keyword TEXT;

CREATE INDEX IF NOT EXISTS idx_lotteries_chat_id ON lotteries (chat_id);
CREATE INDEX IF NOT EXISTS idx_lotteries_status ON lotteries (status);
CREATE INDEX IF NOT EXISTS idx_lotteries_end_at ON lotteries (end_at);
CREATE INDEX IF NOT EXISTS idx_lotteries_created_by ON lotteries (created_by);
CREATE INDEX IF NOT EXISTS idx_lotteries_chat_status_created ON lotteries (chat_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lotteries_due_active ON lotteries (end_at, id)
    WHERE status = 'active' AND end_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_lotteries_active_keyword_lookup ON lotteries (chat_id, created_at DESC)
    WHERE status = 'active' AND join_type IN ('keyword', 'both') AND join_keyword <> '';

CREATE TABLE IF NOT EXISTS lottery_entries (
    id BIGSERIAL PRIMARY KEY,
    lottery_id BIGINT NOT NULL REFERENCES lotteries(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    username VARCHAR(64) NOT NULL DEFAULT '',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_winner BOOLEAN NOT NULL DEFAULT FALSE
);

ALTER TABLE lottery_entries
    ADD COLUMN IF NOT EXISTS username TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_lottery_entry_user ON lottery_entries (lottery_id, user_id);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_lottery_id ON lottery_entries (lottery_id);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_user_id ON lottery_entries (user_id);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_is_winner ON lottery_entries (is_winner);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_lottery_joined ON lottery_entries (lottery_id, joined_at);
CREATE INDEX IF NOT EXISTS idx_lottery_entries_winners ON lottery_entries (lottery_id, joined_at)
    WHERE is_winner = TRUE;

CREATE TABLE IF NOT EXISTS ban_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    reason TEXT,
    banned_by BIGINT,
    banned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unbanned_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ban_logs_user_id ON ban_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_ban_logs_chat_id ON ban_logs (chat_id);
CREATE INDEX IF NOT EXISTS idx_ban_logs_banned_by ON ban_logs (banned_by);

CREATE TABLE IF NOT EXISTS warn_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    reason TEXT,
    warned_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cleared BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_warn_records_user_id ON warn_records (user_id);
CREATE INDEX IF NOT EXISTS idx_warn_records_chat_id ON warn_records (chat_id);
CREATE INDEX IF NOT EXISTS idx_warn_records_warned_by ON warn_records (warned_by);
CREATE INDEX IF NOT EXISTS idx_warn_records_created_at ON warn_records (created_at);
CREATE INDEX IF NOT EXISTS idx_warn_records_cleared ON warn_records (cleared);

CREATE TABLE IF NOT EXISTS chat_admin_configs (
    chat_id BIGINT PRIMARY KEY,
    welcome_text TEXT NOT NULL DEFAULT '欢迎 {name} 加入！',
    verify_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    verify_timeout INTEGER NOT NULL DEFAULT 60,
    warn_limit INTEGER NOT NULL DEFAULT 3,
    updated_at TIMESTAMPTZ
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pin_after_send BOOLEAN NOT NULL DEFAULT FALSE,
    auto_delete_seconds INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE scheduled_posts
    ADD COLUMN IF NOT EXISTS pin_after_send BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS auto_delete_seconds INTEGER NOT NULL DEFAULT 0;

ALTER TABLE level_configs
    ADD COLUMN IF NOT EXISTS badge TEXT NOT NULL DEFAULT '';

ALTER TABLE keyword_filters
    ADD COLUMN IF NOT EXISTS scope TEXT NOT NULL DEFAULT 'chat',
    ADD COLUMN IF NOT EXISTS reply_text TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_scheduled_posts_chat_id ON scheduled_posts (chat_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_enabled ON scheduled_posts (enabled);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_run_once_at ON scheduled_posts (run_once_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_chat_created_at ON scheduled_posts (chat_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_chat_enabled ON scheduled_posts (chat_id, enabled);
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_due_once ON scheduled_posts (run_once_at, id)
    WHERE enabled = TRUE AND last_run_at IS NULL AND run_once_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scheduled_posts_enabled_cron ON scheduled_posts (id)
    WHERE enabled = TRUE AND COALESCE(cron_expr, '') <> '';
CREATE INDEX IF NOT EXISTS idx_keyword_filters_scope ON keyword_filters (scope);
CREATE INDEX IF NOT EXISTS idx_keyword_filters_enabled ON keyword_filters (enabled);
