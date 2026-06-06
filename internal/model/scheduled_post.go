package model

import "time"

type ScheduledPost struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatID    int64  `gorm:"not null;index" json:"chat_id"`
	Title     string `gorm:"size:128" json:"title,omitempty"`
	Content   string `gorm:"type:text" json:"content,omitempty"`
	MediaURL  string `gorm:"type:text" json:"media_url,omitempty"`
	MediaName string `gorm:"type:text;not null;default:''" json:"media_name,omitempty"`
	MediaMime string `gorm:"type:text;not null;default:''" json:"media_mime,omitempty"`
	MediaData []byte `gorm:"type:bytea" json:"-"`
	MediaType string `gorm:"size:16" json:"media_type,omitempty"`

	// ParseMode controls how the caption is rendered. Defaults to "HTML".
	// Also accepts "MarkdownV2" and "" (plain text).
	ParseMode string `gorm:"size:16;not null;default:'HTML'" json:"parse_mode"`

	// InlineKeyboardJSON stores the inline keyboard definition as a JSON
	// array of rows, each row an array of button objects. Example:
	// [ [ {"text":"查看详情","url":"https://..."}, {"text":"我要参与","callback_data":"join"} ] ]
	InlineKeyboardJSON string `gorm:"type:text;not null;default:'[]'" json:"inline_keyboard_json"`

	CronExpr  string `gorm:"size:64" json:"cron_expr,omitempty"`

	RunOnceAt *time.Time `gorm:"index" json:"run_once_at,omitempty"`
	Enabled   bool       `gorm:"not null;default:true;index" json:"enabled"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`

	PinAfterSend      bool `gorm:"not null;default:false" json:"pin_after_send"`
	AutoDeleteSeconds int  `gorm:"not null;default:0" json:"auto_delete_seconds"`
}

func (ScheduledPost) TableName() string {
	return "scheduled_posts"
}
