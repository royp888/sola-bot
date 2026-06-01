CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_user_id BIGINT UNIQUE,
    email TEXT UNIQUE,
    password_hash TEXT,
    username TEXT UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'user',
    language_code TEXT NOT NULL DEFAULT 'zh',
    timezone TEXT NOT NULL DEFAULT 'UTC',
    status TEXT NOT NULL DEFAULT 'active',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    telegram_bot_id BIGINT UNIQUE,
    username TEXT UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    token_ciphertext TEXT,
    status TEXT NOT NULL DEFAULT 'inactive',
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    language_code TEXT NOT NULL DEFAULT 'zh',
    webhook_url TEXT,
    webhook_secret TEXT,
    last_checked_at TIMESTAMPTZ,
    last_error TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS telegram_chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_chat_id BIGINT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    title TEXT,
    username TEXT UNIQUE,
    description TEXT,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    invite_link TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    settings_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS chat_point_configs (
    chat_id BIGINT PRIMARY KEY,
    point_text INTEGER NOT NULL DEFAULT 1,
    point_photo INTEGER NOT NULL DEFAULT 3,
    point_sticker INTEGER NOT NULL DEFAULT 2,
    point_video INTEGER NOT NULL DEFAULT 3,
    point_file INTEGER NOT NULL DEFAULT 2,
    point_voice INTEGER NOT NULL DEFAULT 3,
    cooldown_seconds INTEGER NOT NULL DEFAULT 60,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS user_points (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    total_points BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ,
    CONSTRAINT idx_user_points_user_chat UNIQUE (user_id, chat_id)
);

CREATE TABLE IF NOT EXISTS point_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    delta INTEGER NOT NULL,
    reason VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS chat_admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES telegram_chats(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'admin',
    can_manage BOOLEAN NOT NULL DEFAULT FALSE,
    can_post BOOLEAN NOT NULL DEFAULT FALSE,
    can_delete BOOLEAN NOT NULL DEFAULT FALSE,
    can_ban BOOLEAN NOT NULL DEFAULT FALSE,
    granted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT idx_chat_admin UNIQUE (chat_id, user_id)
);

CREATE TABLE IF NOT EXISTS button_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'global',
    inline_keyboard_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT idx_button_template_owner_name UNIQUE (owner_user_id, name)
);

CREATE TABLE IF NOT EXISTS scheduled_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_key TEXT NOT NULL UNIQUE,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    target_type TEXT NOT NULL,
    target_id UUID,
    chat_id UUID REFERENCES telegram_chats(id) ON DELETE SET NULL,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    cron_expression TEXT,
    run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 0,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_error TEXT,
    locked_at TIMESTAMPTZ,
    lock_owner TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    chat_id UUID REFERENCES telegram_chats(id) ON DELETE SET NULL,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    scheduled_job_id UUID REFERENCES scheduled_jobs(id) ON DELETE SET NULL,
    button_template_id UUID REFERENCES button_templates(id) ON DELETE SET NULL,
    title TEXT,
    content_text TEXT NOT NULL DEFAULT '',
    parse_mode TEXT NOT NULL DEFAULT 'HTML',
    media_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    inline_keyboard_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL DEFAULT 'draft',
    publish_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    telegram_message_id BIGINT,
    telegram_thread_id BIGINT,
    error_message TEXT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'telegram',
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    chat_id UUID REFERENCES telegram_chats(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    post_id UUID REFERENCES posts(id) ON DELETE SET NULL,
    message_id BIGINT,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id UUID REFERENCES telegram_chats(id) ON DELETE SET NULL,
    delta BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    reason TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_id UUID,
    granted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    bot_id UUID REFERENCES bots(id) ON DELETE SET NULL,
    chat_id UUID REFERENCES telegram_chats(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID,
    request_id TEXT,
    ip TEXT,
    user_agent TEXT,
    before_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_status_created_at ON users (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bots_status_created_at ON bots (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_telegram_chats_status_created_at ON telegram_chats (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_point_logs_user_chat ON point_logs (user_id, chat_id);
CREATE INDEX IF NOT EXISTS idx_point_logs_created ON point_logs (created_at);
CREATE INDEX IF NOT EXISTS idx_posts_status_publish_at ON posts (status, publish_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_status_run_at ON scheduled_jobs (status, run_at);
CREATE INDEX IF NOT EXISTS idx_events_type_occurred_at ON events (event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_points_user_recorded_at ON points (user_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_occurred_at ON audit_logs (action, occurred_at DESC);
