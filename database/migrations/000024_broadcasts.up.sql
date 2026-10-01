-- Mass messaging: one-off broadcasts to bot users, managed chats, or both.

CREATE TABLE IF NOT EXISTS broadcasts (
    id                    BIGSERIAL PRIMARY KEY,
    title                 VARCHAR(128) NOT NULL DEFAULT '',
    content               TEXT NOT NULL DEFAULT '',
    media_url             TEXT NOT NULL DEFAULT '',
    media_name            TEXT NOT NULL DEFAULT '',
    media_mime            TEXT NOT NULL DEFAULT '',
    media_data            BYTEA,
    media_type            VARCHAR(16) NOT NULL DEFAULT '',
    parse_mode            VARCHAR(16) NOT NULL DEFAULT 'HTML',
    inline_keyboard_json  TEXT NOT NULL DEFAULT '[]',
    target_type           VARCHAR(16) NOT NULL DEFAULT 'users',
    chat_ids_json         TEXT NOT NULL DEFAULT '[]',
    status                VARCHAR(16) NOT NULL DEFAULT 'draft',
    scheduled_at          TIMESTAMPTZ,
    started_at            TIMESTAMPTZ,
    finished_at           TIMESTAMPTZ,
    total_targets         INTEGER NOT NULL DEFAULT 0,
    sent_count            INTEGER NOT NULL DEFAULT 0,
    failed_count          INTEGER NOT NULL DEFAULT 0,
    last_error            TEXT NOT NULL DEFAULT '',
    created_by_user_id    UUID,
    owner_user_id         UUID,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_broadcasts_status ON broadcasts (status);
CREATE INDEX IF NOT EXISTS idx_broadcasts_target_type ON broadcasts (target_type);
CREATE INDEX IF NOT EXISTS idx_broadcasts_scheduled_at ON broadcasts (scheduled_at);
CREATE INDEX IF NOT EXISTS idx_broadcasts_created_by_user_id ON broadcasts (created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_broadcasts_owner_user_id ON broadcasts (owner_user_id);

CREATE TABLE IF NOT EXISTS broadcast_deliveries (
    id           BIGSERIAL PRIMARY KEY,
    broadcast_id BIGINT NOT NULL REFERENCES broadcasts (id) ON DELETE CASCADE,
    target_type  VARCHAR(16) NOT NULL DEFAULT 'users',
    target_id    BIGINT NOT NULL,
    target_name  VARCHAR(255) NOT NULL DEFAULT '',
    status       VARCHAR(16) NOT NULL DEFAULT 'pending',
    message_id   BIGINT NOT NULL DEFAULT 0,
    error        TEXT NOT NULL DEFAULT '',
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_broadcast_deliveries_broadcast_id ON broadcast_deliveries (broadcast_id);
CREATE INDEX IF NOT EXISTS idx_broadcast_deliveries_status ON broadcast_deliveries (status);
CREATE INDEX IF NOT EXISTS idx_broadcast_deliveries_target_id ON broadcast_deliveries (target_id);
