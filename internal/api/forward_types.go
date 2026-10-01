package api

import (
	"context"
	"time"
)

// AutoForward is the API representation of one 源→目标 转发规则.
type AutoForward struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`

	SourceChatID int64  `json:"source_chat_id"`
	SourceTitle  string `json:"source_title"`
	TargetChatID int64  `json:"target_chat_id"`
	TargetTitle  string `json:"target_title"`

	Mode      string `json:"mode"`
	ParseMode string `json:"parse_mode"`
	Realtime  bool   `json:"realtime"`

	HistoryFromID         int64      `json:"history_from_id"`
	HistoryToID           int64      `json:"history_to_id"`
	HistoryCursor         int64      `json:"history_cursor"`
	HistoryState          string     `json:"history_state"`
	HistorySent           int64      `json:"history_sent"`
	HistoryFailed         int64      `json:"history_failed"`
	HistorySkipped        int64      `json:"history_skipped"`
	HistoryStartedAt      *time.Time `json:"history_started_at,omitempty"`
	HistoryFinishedAt     *time.Time `json:"history_finished_at,omitempty"`
	HistoryAutoStopMisses int        `json:"history_auto_stop_misses"`

	DelayMS    int  `json:"delay_ms"`
	BatchSize  int  `json:"batch_size"`
	AlbumBatch bool `json:"album_batch"`

	SkipKeywords   string `json:"skip_keywords"`
	SkipForwarded  bool   `json:"skip_forwarded"`
	SkipNoMedia    bool   `json:"skip_no_media"`
	SkipNoText     bool   `json:"skip_no_text"`
	SkipDuplicates bool   `json:"skip_duplicates"`
	MinFileSizeMB  int    `json:"min_file_size_mb"`
	MaxFileSizeMB  int    `json:"max_file_size_mb"`
	MinTextLength  int    `json:"min_text_length"`

	CaptionTemplate  string `json:"caption_template"`
	CaptionPrefix    string `json:"caption_prefix"`
	CaptionSuffix    string `json:"caption_suffix"`
	RemoveLinks      bool   `json:"remove_links"`
	RemoveMentions   bool   `json:"remove_mentions"`
	RemoveHashtags   bool   `json:"remove_hashtags"`
	DropSourceLine   bool   `json:"drop_source_line"`
	ReplacementsJSON string `json:"replacements_json"`
	KeepOriginalText bool   `json:"keep_original_text"`

	BuyEnabled       bool   `json:"buy_enabled"`
	BuyText          string `json:"buy_text"`
	BuyURLTemplate   string `json:"buy_url_template"`
	ProductIDRegex   string `json:"product_id_regex"`
	BuyButtonOnMedia bool   `json:"buy_button_on_media"`
	BuyButtonOnText  bool   `json:"buy_button_on_text"`
	ExtraButtonsJSON string `json:"extra_buttons_json"`

	TotalForwarded int64      `json:"total_forwarded"`
	TotalFailed    int64      `json:"total_failed"`
	TotalSkipped   int64      `json:"total_skipped"`
	LastForwardAt  *time.Time `json:"last_forward_at,omitempty"`
	LastError      string     `json:"last_error"`

	OwnerUserID string `json:"owner_user_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AutoForwardCreateRequest creates a rule. StartHistory 为 true 时立即从
// HistoryFromID 开始补发历史。
type AutoForwardCreateRequest struct {
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	SourceChatID int64  `json:"source_chat_id" binding:"required"`
	SourceTitle  string `json:"source_title"`
	TargetChatID int64  `json:"target_chat_id" binding:"required"`
	TargetTitle  string `json:"target_title"`

	Mode      string `json:"mode"`
	ParseMode string `json:"parse_mode"`
	Realtime  *bool  `json:"realtime"`

	HistoryFromID         *int64 `json:"history_from_id"`
	HistoryToID           *int64 `json:"history_to_id"`
	HistoryAutoStopMisses *int   `json:"history_auto_stop_misses"`
	StartHistory          bool   `json:"start_history"`

	DelayMS    *int  `json:"delay_ms"`
	BatchSize  *int  `json:"batch_size"`
	AlbumBatch *bool `json:"album_batch"`

	SkipKeywords   *string `json:"skip_keywords"`
	SkipForwarded  *bool   `json:"skip_forwarded"`
	SkipNoMedia    *bool   `json:"skip_no_media"`
	SkipNoText     *bool   `json:"skip_no_text"`
	SkipDuplicates *bool   `json:"skip_duplicates"`
	MinFileSizeMB  *int    `json:"min_file_size_mb"`
	MaxFileSizeMB  *int    `json:"max_file_size_mb"`
	MinTextLength  *int    `json:"min_text_length"`

	CaptionTemplate  *string `json:"caption_template"`
	CaptionPrefix    *string `json:"caption_prefix"`
	CaptionSuffix    *string `json:"caption_suffix"`
	RemoveLinks      *bool   `json:"remove_links"`
	RemoveMentions   *bool   `json:"remove_mentions"`
	RemoveHashtags   *bool   `json:"remove_hashtags"`
	DropSourceLine   *bool   `json:"drop_source_line"`
	ReplacementsJSON *string `json:"replacements_json"`
	KeepOriginalText *bool   `json:"keep_original_text"`

	BuyEnabled       *bool   `json:"buy_enabled"`
	BuyText          *string `json:"buy_text"`
	BuyURLTemplate   *string `json:"buy_url_template"`
	ProductIDRegex   *string `json:"product_id_regex"`
	BuyButtonOnMedia *bool   `json:"buy_button_on_media"`
	BuyButtonOnText  *bool   `json:"buy_button_on_text"`
	ExtraButtonsJSON *string `json:"extra_buttons_json"`
}

type AutoForwardUpdateRequest = AutoForwardCreateRequest

// AutoForwardLog is one forwarded (or skipped/failed) message.
type AutoForwardLog struct {
	ID              uint64    `json:"id"`
	ForwardID       uint64    `json:"forward_id"`
	RuleName        string    `json:"rule_name"`
	SourceChatID    int64     `json:"source_chat_id"`
	SourceMessageID int64     `json:"source_message_id"`
	TargetChatID    int64     `json:"target_chat_id"`
	TargetMessageID int64     `json:"target_message_id"`
	Status          string    `json:"status"`
	MediaType       string    `json:"media_type"`
	FileSize        int64     `json:"file_size"`
	Text            string    `json:"text"`
	Error           string    `json:"error"`
	CreatedAt       time.Time `json:"created_at"`
}

// AutoForwardPreviewRequest renders a caption + keyboard without sending.
type AutoForwardPreviewRequest struct {
	Text        string `json:"text"`
	HasMedia    bool   `json:"has_media"`
	MediaType   string `json:"media_type"`
	FileSize    int64  `json:"file_size"`
	IsForwarded bool   `json:"is_forwarded"`
	MessageID   int64  `json:"message_id"`
}

// AutoForwardPreviewResponse is the rendered result shown in the UI.
type AutoForwardPreviewResponse struct {
	Caption       string              `json:"caption"`
	Buttons       []InlineKeyboardRow `json:"buttons"`
	BuyURL        string              `json:"buy_url"`
	BuyButtonText string              `json:"buy_button_text"`
	Skipped       bool                `json:"skipped"`
	SkipReason    string              `json:"skip_reason"`
	Warnings      []string            `json:"warnings"`
}

// AutoForwardStats powers the dashboard cards.
type AutoForwardStats struct {
	Total     int64 `json:"total"`
	Enabled   int64 `json:"enabled"`
	Realtime  int64 `json:"realtime"`
	Running   int64 `json:"running"`
	Forwarded int64 `json:"forwarded"`
	Failed    int64 `json:"failed"`
	Skipped   int64 `json:"skipped"`
}

type ForwardService interface {
	Create(ctx context.Context, req AutoForwardCreateRequest, ownerUserID string) (*AutoForward, error)
	List(ctx context.Context, query CommonListQuery, ownerUserID string) ([]AutoForward, error)
	Get(ctx context.Context, id string, ownerUserID string) (*AutoForward, error)
	Update(ctx context.Context, id string, req AutoForwardUpdateRequest, ownerUserID string) (*AutoForward, error)
	Delete(ctx context.Context, id string, ownerUserID string) error

	StartHistory(ctx context.Context, id string, ownerUserID string) (*AutoForward, error)
	PauseHistory(ctx context.Context, id string, ownerUserID string) (*AutoForward, error)
	ResetHistory(ctx context.Context, id string, ownerUserID string) (*AutoForward, error)

	Logs(ctx context.Context, id string, query CommonListQuery, ownerUserID string) ([]AutoForwardLog, error)
	Preview(ctx context.Context, id string, req AutoForwardPreviewRequest, ownerUserID string) (*AutoForwardPreviewResponse, error)
	TestSend(ctx context.Context, id string, req AutoForwardPreviewRequest, ownerUserID string) (*AutoForwardLog, error)
	Stats(ctx context.Context, ownerUserID string) (*AutoForwardStats, error)
}
