package model

type MessageTemplate struct {
	BaseModel

	ChatID    *int64 `gorm:"index" json:"chat_id,omitempty"`
	Name      string `gorm:"type:text;not null" json:"name"`
	Content   string `gorm:"type:text;not null;default:''" json:"content"`
	MediaType string `gorm:"type:text;not null;default:'text'" json:"media_type"`
	MediaURL  string `gorm:"type:text;not null;default:''" json:"media_url"`
	ParseMode string `gorm:"type:text;not null;default:''" json:"parse_mode"`
	CreatedBy int64  `gorm:"index" json:"created_by,omitempty"`
}
