ALTER TABLE scheduled_posts
DROP COLUMN IF EXISTS media_data;

ALTER TABLE scheduled_posts
DROP COLUMN IF EXISTS media_mime;

ALTER TABLE scheduled_posts
DROP COLUMN IF EXISTS media_name;
