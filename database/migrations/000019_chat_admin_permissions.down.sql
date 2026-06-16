ALTER TABLE chat_admins
    DROP COLUMN IF EXISTS can_verify,
    DROP COLUMN IF EXISTS can_keyword,
    DROP COLUMN IF EXISTS can_points;
