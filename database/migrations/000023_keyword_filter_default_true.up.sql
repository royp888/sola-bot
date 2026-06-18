-- Change keyword_filter_enabled default to true so new chats have filtering on by default.
ALTER TABLE chat_moderation_configs
    ALTER COLUMN keyword_filter_enabled SET DEFAULT true;

-- Enable keyword filtering for existing rows that have at least one active keyword but still have it disabled.
UPDATE chat_moderation_configs c
SET keyword_filter_enabled = true
WHERE keyword_filter_enabled = false
  AND EXISTS (
      SELECT 1 FROM keyword_filters k
      WHERE k.chat_id = c.chat_id AND k.enabled = true
  );
