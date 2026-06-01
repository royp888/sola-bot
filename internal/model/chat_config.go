package model

import "time"

type ChatPointConfig struct {
	ChatID          int64     `gorm:"primaryKey" json:"chat_id"`
	PointText       int       `gorm:"not null;default:1" json:"point_text"`
	PointPhoto      int       `gorm:"not null;default:3" json:"point_photo"`
	PointSticker    int       `gorm:"not null;default:2" json:"point_sticker"`
	PointVideo      int       `gorm:"not null;default:3" json:"point_video"`
	PointFile       int       `gorm:"not null;default:2" json:"point_file"`
	PointVoice      int       `gorm:"not null;default:3" json:"point_voice"`
	CooldownSeconds int       `gorm:"not null;default:60" json:"cooldown_seconds"`
	Enabled         bool      `gorm:"not null;default:true" json:"enabled"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserPoint struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64     `gorm:"not null;uniqueIndex:idx_user_points_user_chat,priority:1" json:"user_id"`
	ChatID      int64     `gorm:"not null;uniqueIndex:idx_user_points_user_chat,priority:2;index" json:"chat_id"`
	TotalPoints int64     `gorm:"not null;default:0" json:"total_points"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PointLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null;index:idx_point_logs_user_chat,priority:1" json:"user_id"`
	ChatID    int64     `gorm:"not null;index:idx_point_logs_user_chat,priority:2;index" json:"chat_id"`
	Delta     int       `gorm:"not null" json:"delta"`
	Reason    string    `gorm:"size:64" json:"reason"`
	CreatedAt time.Time `gorm:"not null;default:now();index:idx_point_logs_created" json:"created_at"`
}
