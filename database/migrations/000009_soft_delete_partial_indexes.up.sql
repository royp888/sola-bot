CREATE INDEX IF NOT EXISTS idx_keyword_filters_chat_id_active
    ON keyword_filters (chat_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_violation_records_chat_id_active
    ON violation_records (chat_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_violation_records_user_chat_active
    ON violation_records (user_id, chat_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_auto_replies_chat_id_active
    ON auto_replies (chat_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_message_templates_chat_id_active
    ON message_templates (chat_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_invite_links_chat_id_active
    ON invite_links (chat_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_occurred_at_active
    ON audit_logs (action, occurred_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_chat_admins_chat_id_active
    ON chat_admins (chat_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_button_templates_owner_name_active
    ON button_templates (owner_user_id, name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_level_configs_chat_id_active
    ON level_configs (chat_id)
    WHERE deleted_at IS NULL;
