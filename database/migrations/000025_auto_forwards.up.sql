-- 自动转发：源频道/群 -> 目标频道/群（历史批量 + 实时监听 + 文案改写 + 购买按钮）
CREATE TABLE IF NOT EXISTS auto_forwards (
    id                      BIGSERIAL PRIMARY KEY,
    name                    VARCHAR(128) NOT NULL DEFAULT '',
    enabled                 BOOLEAN      NOT NULL DEFAULT FALSE,

    source_chat_id          BIGINT       NOT NULL DEFAULT 0,
    source_title            VARCHAR(255) NOT NULL DEFAULT '',
    target_chat_id          BIGINT       NOT NULL DEFAULT 0,
    target_title            VARCHAR(255) NOT NULL DEFAULT '',

    mode                    VARCHAR(16)  NOT NULL DEFAULT 'copy',
    parse_mode              VARCHAR(16)  NOT NULL DEFAULT '',
    realtime                BOOLEAN      NOT NULL DEFAULT TRUE,

    history_from_id         BIGINT       NOT NULL DEFAULT 0,
    history_to_id           BIGINT       NOT NULL DEFAULT 0,
    history_cursor          BIGINT       NOT NULL DEFAULT 0,
    history_state           VARCHAR(16)  NOT NULL DEFAULT 'idle',
    history_sent            BIGINT       NOT NULL DEFAULT 0,
    history_failed          BIGINT       NOT NULL DEFAULT 0,
    history_skipped         BIGINT       NOT NULL DEFAULT 0,
    history_started_at      TIMESTAMPTZ,
    history_finished_at     TIMESTAMPTZ,
    history_auto_stop_misses INTEGER     NOT NULL DEFAULT 25,

    delay_ms                INTEGER      NOT NULL DEFAULT 800,
    batch_size              INTEGER      NOT NULL DEFAULT 20,
    album_batch             BOOLEAN      NOT NULL DEFAULT FALSE,

    skip_keywords           TEXT         NOT NULL DEFAULT '',
    skip_forwarded          BOOLEAN      NOT NULL DEFAULT FALSE,
    skip_no_media           BOOLEAN      NOT NULL DEFAULT FALSE,
    skip_no_text            BOOLEAN      NOT NULL DEFAULT FALSE,
    skip_duplicates         BOOLEAN      NOT NULL DEFAULT TRUE,
    min_file_size_mb        INTEGER      NOT NULL DEFAULT 0,
    max_file_size_mb        INTEGER      NOT NULL DEFAULT 0,
    min_text_length         INTEGER      NOT NULL DEFAULT 0,
    block_keywords_note     VARCHAR(255) NOT NULL DEFAULT '',

    caption_template        TEXT         NOT NULL DEFAULT '',
    caption_prefix          TEXT         NOT NULL DEFAULT '',
    caption_suffix          TEXT         NOT NULL DEFAULT '',
    remove_links            BOOLEAN      NOT NULL DEFAULT FALSE,
    remove_mentions         BOOLEAN      NOT NULL DEFAULT FALSE,
    remove_hashtags         BOOLEAN      NOT NULL DEFAULT FALSE,
    drop_source_line        BOOLEAN      NOT NULL DEFAULT FALSE,
    replacements_json       TEXT         NOT NULL DEFAULT '',
    keep_original_text      BOOLEAN      NOT NULL DEFAULT TRUE,

    buy_enabled             BOOLEAN      NOT NULL DEFAULT FALSE,
    buy_text                VARCHAR(64)  NOT NULL DEFAULT '🛒 立即购买',
    buy_url_template        VARCHAR(512) NOT NULL DEFAULT '',
    product_id_regex        VARCHAR(255) NOT NULL DEFAULT '',
    buy_button_on_media     BOOLEAN      NOT NULL DEFAULT TRUE,
    buy_button_on_text      BOOLEAN      NOT NULL DEFAULT TRUE,
    extra_buttons_json      TEXT         NOT NULL DEFAULT '',

    total_forwarded         BIGINT       NOT NULL DEFAULT 0,
    total_failed            BIGINT       NOT NULL DEFAULT 0,
    total_skipped           BIGINT       NOT NULL DEFAULT 0,
    last_forward_at         TIMESTAMPTZ,
    last_error              TEXT         NOT NULL DEFAULT '',

    owner_user_id           UUID,
    created_by_user_id      UUID,

    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auto_forwards_source_chat ON auto_forwards (source_chat_id);
CREATE INDEX IF NOT EXISTS idx_auto_forwards_target_chat ON auto_forwards (target_chat_id);
CREATE INDEX IF NOT EXISTS idx_auto_forwards_history_state ON auto_forwards (history_state);
CREATE INDEX IF NOT EXISTS idx_auto_forwards_owner ON auto_forwards (owner_user_id);
CREATE INDEX IF NOT EXISTS idx_auto_forwards_realtime ON auto_forwards (enabled, realtime, source_chat_id);

CREATE TABLE IF NOT EXISTS auto_forward_logs (
    id                  BIGSERIAL PRIMARY KEY,
    forward_id          BIGINT       NOT NULL,
    rule_name           VARCHAR(128) NOT NULL DEFAULT '',
    source_chat_id      BIGINT       NOT NULL DEFAULT 0,
    source_message_id   BIGINT       NOT NULL DEFAULT 0,
    target_chat_id      BIGINT       NOT NULL DEFAULT 0,
    target_message_id   BIGINT       NOT NULL DEFAULT 0,
    status              VARCHAR(16)  NOT NULL DEFAULT 'sent',
    media_type          VARCHAR(32)  NOT NULL DEFAULT '',
    file_size           BIGINT       NOT NULL DEFAULT 0,
    text                TEXT         NOT NULL DEFAULT '',
    error               TEXT         NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auto_forward_logs_rule_msg ON auto_forward_logs (forward_id, source_chat_id, source_message_id);
CREATE INDEX IF NOT EXISTS idx_auto_forward_logs_forward ON auto_forward_logs (forward_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_auto_forward_logs_status ON auto_forward_logs (status);
