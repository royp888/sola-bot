package api

import (
	"context"
	"time"
)

// Broadcast is the API representation of a mass-messaging job.
type Broadcast struct {
	ID                 uint64     `json:"id"`
	Title              string     `json:"title"`
	Content            string     `json:"content"`
	MediaURL           string     `json:"media_url"`
	MediaName          string     `json:"media_name"`
	MediaType          string     `json:"media_type"`
	HasMedia           bool       `json:"has_media"`
	ParseMode          string     `json:"parse_mode"`
	InlineKeyboardJSON string     `json:"inline_keyboard_json"`
	TargetType         string     `json:"target_type"`
	ChatIDs            []int64    `json:"chat_ids"`
	Status             string     `json:"status"`
	ScheduledAt        *time.Time `json:"scheduled_at,omitempty"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	TotalTargets       int        `json:"total_targets"`
	SentCount          int        `json:"sent_count"`
	FailedCount        int        `json:"failed_count"`
	LastError          string     `json:"last_error"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type BroadcastCreateRequest struct {
	Title              string              `json:"title"`
	Content            string              `json:"content"`
	MediaURL           string              `json:"media_url"`
	MediaName          string              `json:"media_name"`
	MediaMime          string              `json:"media_mime"`
	MediaDataBase64    string              `json:"media_data_base64"`
	MediaType          string              `json:"media_type"`
	ParseMode          string              `json:"parse_mode"`
	InlineKeyboard     []InlineKeyboardRow `json:"inline_keyboard"`
	InlineKeyboardJSON string              `json:"inline_keyboard_json"`
	TargetType         string              `json:"target_type" binding:"required"`
	ChatIDs            []int64             `json:"chat_ids"`
	ScheduledAt        *time.Time          `json:"scheduled_at"`
	SendNow            bool                `json:"send_now"`
}

type BroadcastUpdateRequest struct {
	Title              *string              `json:"title"`
	Content            *string              `json:"content"`
	MediaURL           *string              `json:"media_url"`
	MediaName          *string              `json:"media_name"`
	MediaMime          *string              `json:"media_mime"`
	MediaDataBase64    *string              `json:"media_data_base64"`
	MediaType          *string              `json:"media_type"`
	ParseMode          *string              `json:"parse_mode"`
	InlineKeyboard     *[]InlineKeyboardRow `json:"inline_keyboard"`
	InlineKeyboardJSON *string              `json:"inline_keyboard_json"`
	TargetType         *string              `json:"target_type"`
	ChatIDs            *[]int64             `json:"chat_ids"`
	ScheduledAt        *time.Time           `json:"scheduled_at"`
}

// InlineKeyboardRow / InlineKeyboardButton mirror the post payload shape.
type InlineKeyboardRow struct {
	Buttons []InlineKeyboardButton `json:"buttons"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

// BroadcastDelivery is one target's delivery outcome.
type BroadcastDelivery struct {
	ID         uint64     `json:"id"`
	TargetType string     `json:"target_type"`
	TargetID   int64      `json:"target_id"`
	TargetName string     `json:"target_name"`
	Status     string     `json:"status"`
	MessageID  int64      `json:"message_id"`
	Error      string     `json:"error"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// BroadcastTargetPreview tells the UI how many recipients a draft would reach.
type BroadcastTargetPreview struct {
	Users int64 `json:"users"`
	Chats int64 `json:"chats"`
	Total int64 `json:"total"`
}

// BroadcastStats powers the dashboard cards.
type BroadcastStats struct {
	Total     int64 `json:"total"`
	Draft     int64 `json:"draft"`
	Queued    int64 `json:"queued"`
	Sending   int64 `json:"sending"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
}

type BroadcastService interface {
	Create(ctx context.Context, req BroadcastCreateRequest, ownerUserID string) (*Broadcast, error)
	List(ctx context.Context, query CommonListQuery, ownerUserID string) ([]Broadcast, error)
	Get(ctx context.Context, id string, ownerUserID string) (*Broadcast, error)
	Update(ctx context.Context, id string, req BroadcastUpdateRequest, ownerUserID string) (*Broadcast, error)
	Delete(ctx context.Context, id string, ownerUserID string) error
	Send(ctx context.Context, id string, ownerUserID string) (*Broadcast, error)
	Cancel(ctx context.Context, id string, ownerUserID string) (*Broadcast, error)
	Deliveries(ctx context.Context, id string, query CommonListQuery, ownerUserID string) ([]BroadcastDelivery, error)
	PreviewTargets(ctx context.Context, req BroadcastCreateRequest, ownerUserID string) (*BroadcastTargetPreview, error)
	Stats(ctx context.Context, ownerUserID string) (*BroadcastStats, error)
}
