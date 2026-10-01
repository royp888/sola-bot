package model

import "time"

type ScheduledPostDelivery struct {
	ID              uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ScheduledPostID uint64     `gorm:"not null;index" json:"scheduled_post_id"`
	ChatID          int64      `gorm:"not null;index" json:"chat_id"`
	MessageID       int64      `gorm:"not null" json:"message_id"`
	AutoDeleteAt    *time.Time `gorm:"index" json:"auto_delete_at,omitempty"`
	AutoDeletedAt   *time.Time `gorm:"index" json:"auto_deleted_at,omitempty"`
	// Status: sent / failed；Error 保存失败原因（例如机器人不是该频道管理员）
	Status    string    `gorm:"size:16;not null;default:'sent'" json:"status"`
	Error     string    `gorm:"type:text;not null;default:''" json:"error"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (ScheduledPostDelivery) TableName() string {
	return "scheduled_post_deliveries"
}
