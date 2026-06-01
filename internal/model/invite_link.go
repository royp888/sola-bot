package model

type InviteLink struct {
	BaseModel

	ChatID             int64  `gorm:"not null;index" json:"chat_id"`
	Name               string `gorm:"type:text;not null;default:''" json:"name"`
	InviteLink         string `gorm:"type:text;not null;uniqueIndex:idx_invite_links_link" json:"invite_link"`
	CreatesJoinRequest bool   `gorm:"not null;default:false" json:"creates_join_request"`
	JoinCount          int    `gorm:"not null;default:0" json:"join_count"`
	CreatedBy          int64  `gorm:"index" json:"created_by,omitempty"`
}
