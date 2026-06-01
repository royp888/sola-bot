package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	BaseModel

	TelegramUserID *int64     `gorm:"uniqueIndex" json:"telegram_user_id,omitempty"`
	Email          *string    `gorm:"size:255;uniqueIndex" json:"email,omitempty"`
	PasswordHash   *string    `gorm:"size:255" json:"-"`
	Username       *string    `gorm:"size:64;uniqueIndex" json:"username,omitempty"`
	DisplayName    string     `gorm:"size:255;not null;default:''" json:"display_name"`
	Role           string     `gorm:"size:32;not null;default:'user';index" json:"role"`
	LanguageCode   string     `gorm:"size:16;not null;default:'zh'" json:"language_code"`
	Timezone       string     `gorm:"size:64;not null;default:'UTC'" json:"timezone"`
	Status         string     `gorm:"size:32;not null;default:'active';index" json:"status"`
	IsActive       bool       `gorm:"not null;default:true" json:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	MetadataJSON   string     `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

type Bot struct {
	BaseModel

	OwnerUserID     *UUID      `gorm:"type:uuid;index" json:"owner_user_id,omitempty"`
	TelegramBotID   *int64     `gorm:"uniqueIndex" json:"telegram_bot_id,omitempty"`
	Username        *string    `gorm:"size:64;uniqueIndex" json:"username,omitempty"`
	DisplayName     string     `gorm:"size:255;not null;default:''" json:"display_name"`
	TokenCiphertext *string    `gorm:"size:2048" json:"-"`
	Status          string     `gorm:"size:32;not null;default:'inactive';index" json:"status"`
	IsPrimary       bool       `gorm:"not null;default:false" json:"is_primary"`
	LanguageCode    string     `gorm:"size:16;not null;default:'zh'" json:"language_code"`
	WebhookURL      *string    `gorm:"size:1024" json:"webhook_url,omitempty"`
	WebhookSecret   *string    `gorm:"size:128" json:"webhook_secret,omitempty"`
	LastCheckedAt   *time.Time `json:"last_checked_at,omitempty"`
	LastError       *string    `gorm:"type:text" json:"last_error,omitempty"`
	MetadataJSON    string     `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

type TelegramChat struct {
	BaseModel

	TelegramChatID int64      `gorm:"not null;uniqueIndex" json:"telegram_chat_id"`
	Type           string     `gorm:"size:32;not null;index" json:"type"`
	Title          *string    `gorm:"size:255" json:"title,omitempty"`
	Username       *string    `gorm:"size:255;uniqueIndex" json:"username,omitempty"`
	Description    *string    `gorm:"type:text" json:"description,omitempty"`
	OwnerUserID    *UUID      `gorm:"type:uuid;index" json:"owner_user_id,omitempty"`
	BotID          *UUID      `gorm:"type:uuid;index" json:"bot_id,omitempty"`
	InviteLink     *string    `gorm:"size:1024" json:"invite_link,omitempty"`
	Status         string     `gorm:"size:32;not null;default:'active';index" json:"status"`
	SettingsJSON   string     `gorm:"type:jsonb;not null;default:'{}'" json:"settings_json"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
}

type ChatAdmin struct {
	BaseModel

	ChatID          UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_chat_admin,priority:1" json:"chat_id"`
	UserID          UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_chat_admin,priority:2" json:"user_id"`
	Role            string    `gorm:"size:32;not null;default:'admin';index" json:"role"`
	CanManage       bool      `gorm:"not null;default:false" json:"can_manage"`
	CanPost         bool      `gorm:"not null;default:false" json:"can_post"`
	CanDelete       bool      `gorm:"not null;default:false" json:"can_delete"`
	CanBan          bool      `gorm:"not null;default:false" json:"can_ban"`
	GrantedByUserID *UUID     `gorm:"type:uuid;index" json:"granted_by_user_id,omitempty"`
	GrantedAt       time.Time `gorm:"not null" json:"granted_at"`
	Notes           *string   `gorm:"type:text" json:"notes,omitempty"`
}

type Post struct {
	BaseModel

	BotID              *UUID      `gorm:"type:uuid;index" json:"bot_id,omitempty"`
	ChatID             *UUID      `gorm:"type:uuid;index" json:"chat_id,omitempty"`
	CreatedByUserID    *UUID      `gorm:"type:uuid;index" json:"created_by_user_id,omitempty"`
	ScheduledJobID     *UUID      `gorm:"type:uuid;index" json:"scheduled_job_id,omitempty"`
	ButtonTemplateID   *UUID      `gorm:"type:uuid;index" json:"button_template_id,omitempty"`
	Title              *string    `gorm:"size:255" json:"title,omitempty"`
	ContentText        string     `gorm:"type:text;not null;default:''" json:"content_text"`
	ParseMode          string     `gorm:"size:16;not null;default:'HTML'" json:"parse_mode"`
	MediaJSON          string     `gorm:"type:jsonb;not null;default:'[]'" json:"media_json"`
	InlineKeyboardJSON string     `gorm:"type:jsonb;not null;default:'[]'" json:"inline_keyboard_json"`
	Status             string     `gorm:"size:32;not null;default:'draft';index" json:"status"`
	PublishAt          *time.Time `gorm:"index" json:"publish_at,omitempty"`
	PublishedAt        *time.Time `json:"published_at,omitempty"`
	TelegramMessageID  *int64     `gorm:"index" json:"telegram_message_id,omitempty"`
	TelegramThreadID   *int64     `gorm:"index" json:"telegram_thread_id,omitempty"`
	ErrorMessage       *string    `gorm:"type:text" json:"error_message,omitempty"`
	MetadataJSON       string     `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

type ScheduledJob struct {
	BaseModel

	JobKey          string     `gorm:"size:128;not null;uniqueIndex" json:"job_key"`
	JobType         string     `gorm:"size:64;not null;index" json:"job_type"`
	Status          string     `gorm:"size:32;not null;default:'pending';index" json:"status"`
	TargetType      string     `gorm:"size:64;not null;index" json:"target_type"`
	TargetID        *UUID      `gorm:"type:uuid;index" json:"target_id,omitempty"`
	ChatID          *UUID      `gorm:"type:uuid;index" json:"chat_id,omitempty"`
	BotID           *UUID      `gorm:"type:uuid;index" json:"bot_id,omitempty"`
	CreatedByUserID *UUID      `gorm:"type:uuid;index" json:"created_by_user_id,omitempty"`
	CronExpression  *string    `gorm:"size:128" json:"cron_expression,omitempty"`
	RunAt           *time.Time `gorm:"index" json:"run_at,omitempty"`
	NextRunAt       *time.Time `gorm:"index" json:"next_run_at,omitempty"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	RetryCount      int        `gorm:"not null;default:0" json:"retry_count"`
	MaxRetries      int        `gorm:"not null;default:0" json:"max_retries"`
	PayloadJSON     string     `gorm:"type:jsonb;not null;default:'{}'" json:"payload_json"`
	LastError       *string    `gorm:"type:text" json:"last_error,omitempty"`
	LockedAt        *time.Time `json:"locked_at,omitempty"`
	LockOwner       *string    `gorm:"size:128" json:"lock_owner,omitempty"`
	MetadataJSON    string     `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

type ButtonTemplate struct {
	BaseModel

	OwnerUserID        *UUID   `gorm:"type:uuid;index;uniqueIndex:idx_button_template_owner_name,priority:1" json:"owner_user_id,omitempty"`
	Name               string  `gorm:"size:128;not null;uniqueIndex:idx_button_template_owner_name,priority:2" json:"name"`
	Scope              string  `gorm:"size:32;not null;default:'global';index" json:"scope"`
	InlineKeyboardJSON string  `gorm:"type:jsonb;not null;default:'[]'" json:"inline_keyboard_json"`
	Description        *string `gorm:"type:text" json:"description,omitempty"`
	IsActive           bool    `gorm:"not null;default:true;index" json:"is_active"`
	MetadataJSON       string  `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

type Event struct {
	BaseModel

	EventType    string    `gorm:"size:64;not null;index" json:"event_type"`
	Source       string    `gorm:"size:64;not null;default:'telegram';index" json:"source"`
	BotID        *UUID     `gorm:"type:uuid;index" json:"bot_id,omitempty"`
	ChatID       *UUID     `gorm:"type:uuid;index" json:"chat_id,omitempty"`
	UserID       *UUID     `gorm:"type:uuid;index" json:"user_id,omitempty"`
	PostID       *UUID     `gorm:"type:uuid;index" json:"post_id,omitempty"`
	MessageID    *int64    `gorm:"index" json:"message_id,omitempty"`
	PayloadJSON  string    `gorm:"type:jsonb;not null;default:'{}'" json:"payload_json"`
	OccurredAt   time.Time `gorm:"not null;index" json:"occurred_at"`
	MetadataJSON string    `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

type Point struct {
	BaseModel

	UserID          UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	ChatID          *UUID     `gorm:"type:uuid;index" json:"chat_id,omitempty"`
	Delta           int64     `gorm:"not null" json:"delta"`
	BalanceAfter    int64     `gorm:"not null" json:"balance_after"`
	Reason          string    `gorm:"size:128;not null;index" json:"reason"`
	SourceType      string    `gorm:"size:64;not null;index" json:"source_type"`
	SourceID        *UUID     `gorm:"type:uuid;index" json:"source_id,omitempty"`
	GrantedByUserID *UUID     `gorm:"type:uuid;index" json:"granted_by_user_id,omitempty"`
	RecordedAt      time.Time `gorm:"not null;index" json:"recorded_at"`
	MetadataJSON    string    `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

type AuditLog struct {
	BaseModel

	ActorUserID  *UUID     `gorm:"type:uuid;index" json:"actor_user_id,omitempty"`
	BotID        *UUID     `gorm:"type:uuid;index" json:"bot_id,omitempty"`
	ChatID       *UUID     `gorm:"type:uuid;index" json:"chat_id,omitempty"`
	Action       string    `gorm:"size:128;not null;index" json:"action"`
	EntityType   string    `gorm:"size:64;not null;index" json:"entity_type"`
	EntityID     *UUID     `gorm:"type:uuid;index" json:"entity_id,omitempty"`
	RequestID    *string   `gorm:"size:128;index" json:"request_id,omitempty"`
	IP           *string   `gorm:"size:64" json:"ip,omitempty"`
	UserAgent    *string   `gorm:"type:text" json:"user_agent,omitempty"`
	BeforeJSON   string    `gorm:"type:jsonb;not null;default:'{}'" json:"before_json"`
	AfterJSON    string    `gorm:"type:jsonb;not null;default:'{}'" json:"after_json"`
	MetadataJSON string    `gorm:"type:jsonb;not null;default:'{}'" json:"metadata_json"`
	OccurredAt   time.Time `gorm:"not null;index" json:"occurred_at"`
}

// UUID is a local alias to keep the entity declarations compact.
type UUID = uuid.UUID
