package model

import "time"

type Lottery struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatID          int64      `gorm:"not null;index" json:"chat_id"`
	Title           string     `gorm:"size:128" json:"title"`
	Prize           string     `gorm:"type:text" json:"prize"`
	CostPoints      int        `gorm:"not null;default:0" json:"cost_points"`
	MaxParticipants int        `gorm:"not null;default:0" json:"max_participants"`
	WinnerCount     int        `gorm:"not null;default:1" json:"winner_count"`
	EndAt           *time.Time `gorm:"index" json:"end_at,omitempty"`
	Status          string     `gorm:"size:16;not null;default:'active';index" json:"status"`
	JoinType        string     `gorm:"size:16;not null;default:'button'" json:"join_type"`
	JoinKeyword     string     `gorm:"size:64" json:"join_keyword,omitempty"`
	CreatedBy       int64      `gorm:"index" json:"created_by"`
	CreatedAt       time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

type LotteryEntry struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	LotteryID int64     `gorm:"not null;uniqueIndex:idx_lottery_entry_user,priority:1;index" json:"lottery_id"`
	UserID    int64     `gorm:"not null;uniqueIndex:idx_lottery_entry_user,priority:2;index" json:"user_id"`
	Username  string    `gorm:"size:64" json:"username,omitempty"`
	JoinedAt  time.Time `gorm:"not null;default:now()" json:"joined_at"`
	IsWinner  bool      `gorm:"not null;default:false;index" json:"is_winner"`
}
