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
	ChatID             int64     `gorm:"primaryKey" json:"chat_id"`
	WelcomeText        string    `gorm:"type:text;not null;default:'欢迎 {name} 加入！'" json:"welcome_text"`
	VerifyEnabled      bool      `gorm:"not null;default:true" json:"verify_enabled"`
	VerifyType         string    `gorm:"type:text;not null;default:'button'" json:"verify_type"`
	VerifyTimeout      int       `gorm:"not null;default:60" json:"verify_timeout"`
	VerifyQuestion     string    `gorm:"type:text;not null;default:''" json:"verify_question"`
	VerifyOptions      string    `gorm:"type:text;not null;default:'[]'" json:"verify_options"`
	VerifyCorrectIndex int       `gorm:"not null;default:-1" json:"verify_correct_index"`
	VerifyWhitelist    string    `gorm:"type:text;not null;default:''" json:"verify_whitelist"`
	VerifyDifficulty   string    `gorm:"type:text;not null;default:'medium'" json:"verify_difficulty"`
	WarnLimit          int       `gorm:"not null;default:3" json:"warn_limit"`
	RulesText          string    `gorm:"type:text;not null;default:''" json:"rules_text"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type SeenUser struct {
	ChatID int64     `gorm:"primaryKey;autoIncrement:false" json:"chat_id"`
	UserID int64     `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	SeenAt time.Time `gorm:"not null;default:now()" json:"seen_at"`
}
