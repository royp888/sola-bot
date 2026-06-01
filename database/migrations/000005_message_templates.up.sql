CREATE TABLE IF NOT EXISTS message_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id BIGINT,
    name TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    media_type TEXT NOT NULL DEFAULT 'text',
    media_url TEXT NOT NULL DEFAULT '',
    parse_mode TEXT NOT NULL DEFAULT '',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_message_templates_chat_id ON message_templates (chat_id);
CREATE INDEX IF NOT EXISTS idx_message_templates_created_by ON message_templates (created_by);
