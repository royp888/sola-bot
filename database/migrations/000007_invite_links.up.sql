CREATE TABLE IF NOT EXISTS invite_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id BIGINT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    invite_link TEXT NOT NULL,
    creates_join_request BOOLEAN NOT NULL DEFAULT FALSE,
    join_count INTEGER NOT NULL DEFAULT 0,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_invite_links_chat_id ON invite_links (chat_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invite_links_link ON invite_links (invite_link);
