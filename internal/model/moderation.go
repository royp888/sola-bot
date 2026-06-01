package model

import "time"

type LevelConfig struct {
	BaseModel

	ChatID       int64  `gorm:"not null;uniqueIndex:idx_level_configs_chat_level,priority:1;index" json:"chat_id"`
	Level        int    `gorm:"not null;uniqueIndex:idx_level_configs_chat_level,priority:2" json:"level"`
	MinPoints    int64  `gorm:"not null;default:0" json:"min_points"`
	Label        string `gorm:"type:text;not null;default:''" json:"label"`
	Badge        string `gorm:"type:text;not null;default:''" json:"badge"`
	CanPostLink  bool   `gorm:"not null;default:true" json:"can_post_link"`
	CanPostMedia bool   `gorm:"not null;default:true" json:"can_post_media"`
}

type ViolationRecord struct {
	BaseModel

	UserID          int64  `gorm:"not null;index:idx_violation_records_user_chat,priority:1" json:"user_id"`
	ChatID          int64  `gorm:"not null;index:idx_violation_records_user_chat,priority:2;index" json:"chat_id"`
	ViolationType   string `gorm:"type:text;not null;index" json:"violation_type"`
	ActionTaken     string `gorm:"type:text;not null;index" json:"action_taken"`
	MessageText     string `gorm:"type:text" json:"message_text,omitempty"`
	DetectedBy      string `gorm:"type:text;not null;default:'rule';index" json:"detected_by"`
	DurationSeconds int    `gorm:"default:0" json:"duration_seconds,omitempty"`
	Cleared         bool   `gorm:"not null;default:false;index" json:"cleared"`
}

type ChatModerationConfig struct {
	ChatID               int64     `gorm:"primaryKey" json:"chat_id"`
	VerifyEnabled        bool      `gorm:"not null;default:true" json:"verify_enabled"`
	VerifyType           string    `gorm:"type:text;not null;default:'button'" json:"verify_type"`
	VerifyTimeoutSeconds int       `gorm:"not null;default:60" json:"verify_timeout_seconds"`
	WarnLimit            int       `gorm:"not null;default:3" json:"warn_limit"`
	BlockLinks           bool      `gorm:"not null;default:false" json:"block_links"`
	BlockForwards        bool      `gorm:"not null;default:false" json:"block_forwards"`
	BlockMedia           bool      `gorm:"not null;default:false" json:"block_media"`
	KeywordFilterEnabled bool      `gorm:"not null;default:false" json:"keyword_filter_enabled"`
	SpamScoreThreshold   int       `gorm:"not null;default:60" json:"spam_score_threshold"`
	WelcomeText          string    `gorm:"type:text;not null;default:'欢迎 {name}！'" json:"welcome_text"`
	WelcomeDeleteSeconds int       `gorm:"not null;default:30" json:"welcome_delete_seconds"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type KeywordFilter struct {
	BaseModel

	ChatID    int64  `gorm:"not null;uniqueIndex:idx_keyword_filters_chat_keyword,priority:1;index" json:"chat_id"`
	Keyword   string `gorm:"type:text;not null;uniqueIndex:idx_keyword_filters_chat_keyword,priority:2" json:"keyword"`
	MatchType string `gorm:"type:text;not null;default:'contains'" json:"match_type"`
	Action    string `gorm:"type:text;not null;default:'delete'" json:"action"`
	Scope     string `gorm:"type:text;not null;default:'chat';index" json:"scope"`
	ReplyText string `gorm:"type:text;not null;default:''" json:"reply_text,omitempty"`
	Enabled   bool   `gorm:"not null;default:true;index" json:"enabled"`
	CreatedBy int64  `gorm:"index" json:"created_by,omitempty"`
}
