package model

import "time"

type BanLog struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64      `gorm:"not null;index" json:"user_id"`
	ChatID     int64      `gorm:"not null;index" json:"chat_id"`
	Reason     string     `gorm:"type:text" json:"reason"`
	BannedBy   int64      `gorm:"index" json:"banned_by"`
	BannedAt   time.Time  `gorm:"not null;default:now()" json:"banned_at"`
	UnbannedAt *time.Time `json:"unbanned_at,omitempty"`
}

type WarnRecord struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null;index" json:"user_id"`
	ChatID    int64     `gorm:"not null;index" json:"chat_id"`
	Reason    string    `gorm:"type:text" json:"reason"`
	WarnedBy  int64     `gorm:"index" json:"warned_by"`
	CreatedAt time.Time `gorm:"not null;default:now();index" json:"created_at"`
	Cleared   bool      `gorm:"not null;default:false;index" json:"cleared"`
}

type ChatAdminConfig struct {
	ChatID        int64     `gorm:"primaryKey" json:"chat_id"`
	WelcomeText   string    `gorm:"type:text;not null;default:'欢迎 {name} 加入！'" json:"welcome_text"`
	VerifyEnabled bool      `gorm:"not null;default:true" json:"verify_enabled"`
	VerifyTimeout int       `gorm:"not null;default:60" json:"verify_timeout"`
	WarnLimit     int       `gorm:"not null;default:3" json:"warn_limit"`
	UpdatedAt     time.Time `json:"updated_at"`
}
