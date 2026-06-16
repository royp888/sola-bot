-- Migration 000018: fields for user-contributed features
-- Covers: Turnstile second verify, goodbye, content locks,
--         auto-delete bot messages, rules text, antiflood

ALTER TABLE chat_admin_configs
ADD COLUMN IF NOT EXISTS verify_second_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS goodbye_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS goodbye_text            TEXT    NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS auto_delete_bot_msg_sec INTEGER NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS rules_text              TEXT    NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS lock_links              BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS lock_media              BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS lock_forward            BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS lock_sticker            BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS lock_voice              BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS lock_gif                BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS antiflood_enabled       BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS antiflood_window_sec    INTEGER NOT NULL DEFAULT 10,
ADD COLUMN IF NOT EXISTS antiflood_max_messages  INTEGER NOT NULL DEFAULT 5,
ADD COLUMN IF NOT EXISTS antiflood_action        TEXT    NOT NULL DEFAULT 'mute';
