ALTER TABLE scheduled_posts
DROP COLUMN IF EXISTS inline_keyboard_json;

ALTER TABLE scheduled_posts
DROP COLUMN IF EXISTS parse_mode;
