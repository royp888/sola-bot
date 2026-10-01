package model

import (
	"time"

	"github.com/google/uuid"
)

// 转发模式
const (
	// ForwardModeCopy 去来源复制：可改写文案、可加购买按钮（推荐）
	ForwardModeCopy = "copy"
	// ForwardModeForward 原生转发：保留“转发自”来源标签，无法改文案/按钮
	ForwardModeForward = "forward"
)

// 历史批量转发的运行状态
const (
	ForwardStateIdle      = "idle"
	ForwardStateRunning   = "running"
	ForwardStatePaused    = "paused"
	ForwardStateCompleted = "completed"
	ForwardStateFailed    = "failed"
)

// 单条转发记录状态
const (
	ForwardLogSent    = "sent"
	ForwardLogFailed  = "failed"
	ForwardLogSkipped = "skipped"
)

// AutoForward 自动转发规则：源频道/群 → 目标频道/群。
// 支持历史批量补发（HistoryFromID..HistoryToID）与实时监听（Realtime）。
type AutoForward struct {
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	Name    string `gorm:"size:128;not null;default:''" json:"name"`
	Enabled bool   `gorm:"not null;default:false" json:"enabled"`

	SourceChatID int64  `gorm:"index;not null;default:0" json:"source_chat_id"`
	SourceTitle  string `gorm:"size:255;not null;default:''" json:"source_title"`
	TargetChatID int64  `gorm:"index;not null;default:0" json:"target_chat_id"`
	TargetTitle  string `gorm:"size:255;not null;default:''" json:"target_title"`

	// Mode: copy / forward
	Mode string `gorm:"size:16;not null;default:'copy'" json:"mode"`
	// ParseMode 仅作用于改写后的文案: HTML / MarkdownV2 / 空字符串
	ParseMode string `gorm:"size:16;not null;default:''" json:"parse_mode"`
	// Realtime 是否实时监听新消息
	Realtime bool `gorm:"not null;default:true" json:"realtime"`

	// 历史批量转发
	HistoryFromID     int64      `gorm:"not null;default:0" json:"history_from_id"`
	HistoryToID       int64      `gorm:"not null;default:0" json:"history_to_id"`
	HistoryCursor     int64      `gorm:"not null;default:0" json:"history_cursor"`
	HistoryState      string     `gorm:"size:16;not null;default:'idle';index" json:"history_state"`
	HistorySent       int64      `gorm:"not null;default:0" json:"history_sent"`
	HistoryFailed     int64      `gorm:"not null;default:0" json:"history_failed"`
	HistorySkipped    int64      `gorm:"not null;default:0" json:"history_skipped"`
	HistoryStartedAt  *time.Time `json:"history_started_at"`
	HistoryFinishedAt *time.Time `json:"history_finished_at"`
	// HistoryAutoStopMisses 连续多少条“消息不存在”后判定历史区间结束（自动探测尾部）
	HistoryAutoStopMisses int `gorm:"not null;default:25" json:"history_auto_stop_misses"`

	// DelayMS 每条之间的间隔，防风控
	DelayMS int `gorm:"not null;default:800" json:"delay_ms"`
	// BatchSize 每轮处理条数
	BatchSize int `gorm:"not null;default:20" json:"batch_size"`

	// 相册合并：用 copyMessages 保留相册（此时无法改文案/按钮）
	AlbumBatch bool `gorm:"not null;default:false" json:"album_batch"`

	// 过滤条件
	SkipKeywords      string `gorm:"type:text;not null;default:''" json:"skip_keywords"`
	SkipForwarded     bool   `gorm:"not null;default:false" json:"skip_forwarded"`
	SkipNoMedia       bool   `gorm:"not null;default:false" json:"skip_no_media"`
	SkipNoText        bool   `gorm:"not null;default:false" json:"skip_no_text"`
	SkipDuplicates    bool   `gorm:"not null;default:true" json:"skip_duplicates"`
	MinFileSizeMB     int    `gorm:"not null;default:0" json:"min_file_size_mb"`
	MaxFileSizeMB     int    `gorm:"not null;default:0" json:"max_file_size_mb"`
	MinTextLength     int    `gorm:"not null;default:0" json:"min_text_length"`
	BlockKeywordsNote string `gorm:"size:255;not null;default:''" json:"block_keywords_note"`

	// 文案改写
	CaptionTemplate  string `gorm:"type:text;not null;default:''" json:"caption_template"`
	CaptionPrefix    string `gorm:"type:text;not null;default:''" json:"caption_prefix"`
	CaptionSuffix    string `gorm:"type:text;not null;default:''" json:"caption_suffix"`
	RemoveLinks      bool   `gorm:"not null;default:false" json:"remove_links"`
	RemoveMentions   bool   `gorm:"not null;default:false" json:"remove_mentions"`
	RemoveHashtags   bool   `gorm:"not null;default:false" json:"remove_hashtags"`
	DropSourceLine   bool   `gorm:"not null;default:false" json:"drop_source_line"`
	ReplacementsJSON string `gorm:"type:text;not null;default:''" json:"replacements_json"`
	// KeepOriginalText 未命中改写模板时是否保留原文（false 表示只发前缀+后缀）
	KeepOriginalText bool `gorm:"not null;default:true" json:"keep_original_text"`

	// 购买按钮
	BuyEnabled       bool   `gorm:"not null;default:false" json:"buy_enabled"`
	BuyText          string `gorm:"size:64;not null;default:'🛒 立即购买'" json:"buy_text"`
	BuyURLTemplate   string `gorm:"size:512;not null;default:''" json:"buy_url_template"`
	ProductIDRegex   string `gorm:"size:255;not null;default:''" json:"product_id_regex"`
	BuyButtonOnMedia bool   `gorm:"not null;default:true" json:"buy_button_on_media"`
	BuyButtonOnText  bool   `gorm:"not null;default:true" json:"buy_button_on_text"`
	ExtraButtonsJSON string `gorm:"type:text;not null;default:''" json:"extra_buttons_json"`

	// 统计
	TotalForwarded int64      `gorm:"not null;default:0" json:"total_forwarded"`
	TotalFailed    int64      `gorm:"not null;default:0" json:"total_failed"`
	TotalSkipped   int64      `gorm:"not null;default:0" json:"total_skipped"`
	LastForwardAt  *time.Time `json:"last_forward_at"`
	LastError      string     `gorm:"type:text;not null;default:''" json:"last_error"`

	OwnerUserID     *uuid.UUID `gorm:"type:uuid;index" json:"owner_user_id"`
	CreatedByUserID *uuid.UUID `gorm:"type:uuid" json:"created_by_user_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AutoForward) TableName() string { return "auto_forwards" }

// AutoForwardLog 单条转发记录，用于断点续传与去重。
type AutoForwardLog struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ForwardID uint64 `gorm:"not null;index:idx_auto_forward_logs_rule_msg,unique" json:"forward_id"`
	RuleName  string `gorm:"size:128;not null;default:''" json:"rule_name"`

	SourceChatID    int64 `gorm:"not null;default:0;index:idx_auto_forward_logs_rule_msg,unique" json:"source_chat_id"`
	SourceMessageID int64 `gorm:"not null;default:0;index:idx_auto_forward_logs_rule_msg,unique" json:"source_message_id"`
	TargetChatID    int64 `gorm:"not null;default:0" json:"target_chat_id"`
	TargetMessageID int64 `gorm:"not null;default:0" json:"target_message_id"`

	Status    string `gorm:"size:16;not null;default:'sent';index" json:"status"`
	MediaType string `gorm:"size:32;not null;default:''" json:"media_type"`
	FileSize  int64  `gorm:"not null;default:0" json:"file_size"`
	Text      string `gorm:"type:text;not null;default:''" json:"text"`
	Error     string `gorm:"type:text;not null;default:''" json:"error"`

	CreatedAt time.Time `json:"created_at"`
}

func (AutoForwardLog) TableName() string { return "auto_forward_logs" }
