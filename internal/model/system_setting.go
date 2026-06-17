package model

import "time"

// SystemSetting is a key-value store for runtime-configurable settings.
// Settings here override values from the environment / config file without requiring a restart.
type SystemSetting struct {
	Key       string    `gorm:"primaryKey;type:varchar(128)" json:"key"`
	Value     string    `gorm:"type:text;not null;default:''" json:"value"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}
