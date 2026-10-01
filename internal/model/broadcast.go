package model

import "time"

// Broadcast target types.
const (
	BroadcastTargetUsers  = "users"
	BroadcastTargetGroups = "groups"
	BroadcastTargetBoth   = "both"
)

// Broadcast lifecycle states.
const (
	BroadcastStatusDraft     = "draft"
	BroadcastStatusQueued    = "queued"
	BroadcastStatusSending   = "sending"
	BroadcastStatusCompleted = "completed"
	BroadcastStatusFailed    = "failed"
	BroadcastStatusCanceled  = "canceled"
)

// BroadcastDelivery states.
const (
	BroadcastDeliveryPending = "pending"
	BroadcastDeliverySent    = "sent"
	BroadcastDeliveryFailed  = "failed"
)

// Broadcast is a one-off mass message: it fans a single piece of content out
// to every user of the bot, to every chat the bot manages, or both.
type Broadcast struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	Title   string `gorm:"size:128;not null;default:''" json:"title"`
	Content string `gorm:"type:text" json:"content"`

	// Media is stored inline (bytea) exactly like ScheduledPost, so the worker
	// can upload it without needing an external file server.
	MediaURL  string `gorm:"type:text;not null;default:''" json:"media_url"`
	MediaName string `gorm:"type:text;not null;default:''" json:"media_name"`
	MediaMime string `gorm:"type:text;not null;default:''" json:"media_mime"`
	MediaData []byte `gorm:"type:bytea" json:"-"`
	MediaType string `gorm:"size:16;not null;default:''" json:"media_type"`

	// ParseMode: HTML | MarkdownV2 | "" (plain text)
	ParseMode string `gorm:"size:16;not null;default:'HTML'" json:"parse_mode"`

	// InlineKeyboardJSON mirrors ScheduledPost: array of rows, each row an
	// array of button objects.
	InlineKeyboardJSON string `gorm:"type:text;not null;default:'[]'" json:"inline_keyboard_json"`

	// TargetType is one of users | groups | both.
	TargetType string `gorm:"size:16;not null;default:'users';index" json:"target_type"`
	// ChatIDsJSON optionally narrows a groups/both broadcast to a subset of
	// chats. An empty array means "every active chat".
	ChatIDsJSON string `gorm:"type:text;not null;default:'[]'" json:"chat_ids_json"`

	Status string `gorm:"size:16;not null;default:'draft';index" json:"status"`

	ScheduledAt *time.Time `gorm:"index" json:"scheduled_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`

	TotalTargets int    `gorm:"not null;default:0" json:"total_targets"`
	SentCount    int    `gorm:"not null;default:0" json:"sent_count"`
	FailedCount  int    `gorm:"not null;default:0" json:"failed_count"`
	LastError    string `gorm:"type:text;not null;default:''" json:"last_error"`

	CreatedByUserID *UUID     `gorm:"type:uuid;index" json:"created_by_user_id,omitempty"`
	OwnerUserID     *UUID     `gorm:"type:uuid;index" json:"owner_user_id,omitempty"`
	CreatedAt       time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (Broadcast) TableName() string { return "broadcasts" }

// BroadcastDelivery records the outcome of one broadcast for one target.
type BroadcastDelivery struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	BroadcastID uint64 `gorm:"not null;index" json:"broadcast_id"`

	TargetType string `gorm:"size:16;not null;default:'users'" json:"target_type"`
	TargetID   int64  `gorm:"not null;index" json:"target_id"`
	TargetName string `gorm:"size:255;not null;default:''" json:"target_name"`

	Status    string `gorm:"size:16;not null;default:'pending';index" json:"status"`
	MessageID int64  `gorm:"not null;default:0" json:"message_id"`
	Error     string `gorm:"type:text;not null;default:''" json:"error"`

	SentAt    *time.Time `json:"sent_at,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (BroadcastDelivery) TableName() string { return "broadcast_deliveries" }
