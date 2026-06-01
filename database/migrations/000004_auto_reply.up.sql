CREATE TABLE IF NOT EXISTS auto_replies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id BIGINT NOT NULL,
    keyword TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'contains',
    reply_text TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT idx_auto_replies_chat_keyword UNIQUE (chat_id, keyword)
);

CREATE INDEX IF NOT EXISTS idx_auto_replies_chat_id ON auto_replies (chat_id);
CREATE INDEX IF NOT EXISTS idx_auto_replies_enabled ON auto_replies (enabled);
CREATE INDEX IF NOT EXISTS idx_auto_replies_created_by ON auto_replies (created_by);
