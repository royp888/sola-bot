package model

type AutoReply struct {
	BaseModel

	ChatID    int64  `gorm:"not null;uniqueIndex:idx_auto_replies_chat_keyword,priority:1;index" json:"chat_id"`
	Keyword   string `gorm:"type:text;not null;uniqueIndex:idx_auto_replies_chat_keyword,priority:2" json:"keyword"`
	MatchType string `gorm:"type:text;not null;default:'contains'" json:"match_type"`
	ReplyText string `gorm:"type:text;not null;default:''" json:"reply_text"`
	Enabled   bool   `gorm:"not null;index" json:"enabled"`
	CreatedBy int64  `gorm:"index" json:"created_by,omitempty"`
}
