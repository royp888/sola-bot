package api

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

const (
	// ContextKeyAuthClaims stores *AdminClaims in gin.Context.
	ContextKeyAuthClaims = "auth.claims"
)

var ErrForbidden = errors.New("forbidden")

// JWTConfig controls signing and verification for admin sessions.
type JWTConfig struct {
	SigningKey string
	Issuer     string
	TTL        time.Duration
}

type LoginRateLimiter interface {
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type Dependencies struct {
	Auth             AdminAuthService
	BotConfig        BotConfigService
	Chats            ChatBindingService
	ChatPointConfigs ChatPointConfigService
	Points           PointsAdminService
	Admin            ChatAdminService
	Lotteries        LotteryService
	Levels           LevelService
	AdminViolations  AdminViolationService
	Keywords         KeywordService
	AutoReplies      AutoReplyService
	Backups          BackupService
	Templates        TemplateService
	InviteLinks      InviteLinkService
	Posts            PostService
	Schedules        ScheduleService
	Stats            StatsService
	Users            UserService
	JWT              JWTConfig
	Redis            LoginRateLimiter
	AllowedOriginSet []string
	EnableSwagger    bool
}

func (d Dependencies) AllowedOrigins() []string {
	return d.AllowedOriginSet
}

type AdminClaims struct {
	AdminID        string `json:"admin_id"`
	UserID         string `json:"user_id,omitempty"`
	TelegramUserID int64  `json:"telegram_user_id,omitempty"`
	Username       string `json:"username"`
	Role           string `json:"role"`
	jwt.RegisteredClaims
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

type TelegramLoginRequest map[string]any

type AdminIdentity struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id,omitempty"`
	TelegramUserID int64  `json:"telegram_user_id,omitempty"`
	Username       string `json:"username"`
	Email          string `json:"email,omitempty"`
	Role           string `json:"role"`
	Name           string `json:"name,omitempty"`
	DisplayName    string `json:"display_name,omitempty"`
	PhotoURL       string `json:"photo_url,omitempty"`
	Language       string `json:"language,omitempty"`
}

type AdminLoginResponse struct {
	AccessToken string        `json:"accessToken"`
	TokenType   string        `json:"tokenType"`
	ExpiresAt   time.Time     `json:"expiresAt"`
	User        AdminIdentity `json:"user"`
	Admin       AdminIdentity `json:"admin,omitempty"`
}

type HealthResponse struct {
	OK        bool      `json:"ok"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

type BotConfig struct {
	Enabled             bool   `json:"enabled"`
	DefaultLanguage     string `json:"default_language"`
	TimeZone            string `json:"time_zone"`
	AutoDeleteEnabled   bool   `json:"auto_delete_enabled"`
	AutoDeleteAfterSecs int    `json:"auto_delete_after_secs"`
	AllowForwardedPosts bool   `json:"allow_forwarded_posts"`
	EnableStatsTracking bool   `json:"enable_stats_tracking"`
	EnablePoints        bool   `json:"enable_points"`
}

type BotConfigUpdateRequest struct {
	Enabled             *bool   `json:"enabled,omitempty"`
	DefaultLanguage     *string `json:"default_language,omitempty"`
	TimeZone            *string `json:"time_zone,omitempty"`
	AutoDeleteEnabled   *bool   `json:"auto_delete_enabled,omitempty"`
	AutoDeleteAfterSecs *int    `json:"auto_delete_after_secs,omitempty"`
	AllowForwardedPosts *bool   `json:"allow_forwarded_posts,omitempty"`
	EnableStatsTracking *bool   `json:"enable_stats_tracking,omitempty"`
	EnablePoints        *bool   `json:"enable_points,omitempty"`
}

type ChatBindingRequest struct {
	ChatID              int64  `json:"chat_id" binding:"required"`
	ChatType            string `json:"chat_type" binding:"required"`
	Title               string `json:"title"`
	Username            string `json:"username,omitempty"`
	InviteLink          string `json:"invite_link,omitempty"`
	BoundBy             string `json:"bound_by,omitempty"`
	Description         string `json:"description,omitempty"`
	OwnerTelegramUserID int64  `json:"owner_telegram_user_id,omitempty"`
	OwnerUsername       string `json:"owner_username,omitempty"`
	OwnerDisplayName    string `json:"owner_display_name,omitempty"`
}

type ChatBinding struct {
	ChatID      int64     `json:"chat_id"`
	ChatType    string    `json:"chat_type"`
	Title       string    `json:"title"`
	Username    string    `json:"username,omitempty"`
	InviteLink  string    `json:"invite_link,omitempty"`
	BoundBy     string    `json:"bound_by,omitempty"`
	Description string    `json:"description,omitempty"`
	OwnerUserID string    `json:"owner_user_id,omitempty"`
	BoundAt     time.Time `json:"bound_at"`
}

type ChatPointConfig struct {
	ChatID          int64 `json:"chat_id"`
	PointText       int   `json:"point_text"`
	PointPhoto      int   `json:"point_photo"`
	PointSticker    int   `json:"point_sticker"`
	PointVideo      int   `json:"point_video"`
	PointFile       int   `json:"point_file"`
	PointVoice      int   `json:"point_voice"`
	CooldownSeconds int   `json:"cooldown_seconds"`
	Enabled         bool  `json:"enabled"`
}

type ChatPointConfigUpdateRequest struct {
	PointText       *int  `json:"point_text,omitempty" binding:"omitempty,gte=0"`
	PointPhoto      *int  `json:"point_photo,omitempty" binding:"omitempty,gte=0"`
	PointSticker    *int  `json:"point_sticker,omitempty" binding:"omitempty,gte=0"`
	PointVideo      *int  `json:"point_video,omitempty" binding:"omitempty,gte=0"`
	PointFile       *int  `json:"point_file,omitempty" binding:"omitempty,gte=0"`
	PointVoice      *int  `json:"point_voice,omitempty" binding:"omitempty,gte=0"`
	CooldownSeconds *int  `json:"cooldown_seconds,omitempty" binding:"omitempty,gte=0"`
	Enabled         *bool `json:"enabled,omitempty"`
}

type PointRankQuery struct {
	Period string `form:"period,default=all"`
	Limit  int    `form:"limit,default=10" binding:"gte=1,lte=100"`
}

type PointRankItem struct {
	Rank   int   `json:"rank"`
	UserID int64 `json:"user_id"`
	Points int64 `json:"points"`
}

type PointUserResponse struct {
	UserID      int64     `json:"user_id"`
	ChatID      int64     `json:"chat_id"`
	TotalPoints int64     `json:"total_points"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PointUserAdjustRequest struct {
	Delta  int    `json:"delta" binding:"required"`
	Reason string `json:"reason,omitempty"`
}

type PointLogItem struct {
	ID        uint64    `json:"id"`
	UserID    int64     `json:"user_id"`
	ChatID    int64     `json:"chat_id"`
	Delta     int       `json:"delta"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PointLogListQuery struct {
	Limit       int    `form:"limit,default=20" binding:"gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"gte=0"`
	Cursor      string `form:"cursor"`
	OwnerUserID string `form:"-"`
}

type PointLogListResponse struct {
	Items      []PointLogItem `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type CursorListResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type ChatAdminConfig struct {
	ChatID        int64  `json:"chat_id"`
	WelcomeText   string `json:"welcome_text"`
	VerifyEnabled bool   `json:"verify_enabled"`
	VerifyTimeout int    `json:"verify_timeout"`
	WarnLimit     int    `json:"warn_limit"`
}

type ChatAdminConfigUpdateRequest struct {
	WelcomeText   *string `json:"welcome_text,omitempty"`
	VerifyEnabled *bool   `json:"verify_enabled,omitempty"`
	VerifyTimeout *int    `json:"verify_timeout,omitempty" binding:"omitempty,gte=0"`
	WarnLimit     *int    `json:"warn_limit,omitempty" binding:"omitempty,gte=1"`
}

type BanLog struct {
	ID         uint64     `json:"id"`
	UserID     int64      `json:"user_id"`
	ChatID     int64      `json:"chat_id"`
	Reason     string     `json:"reason,omitempty"`
	BannedBy   int64      `json:"banned_by,omitempty"`
	BannedAt   time.Time  `json:"banned_at"`
	UnbannedAt *time.Time `json:"unbanned_at,omitempty"`
}

type WarnRecord struct {
	ID        uint64    `json:"id"`
	UserID    int64     `json:"user_id"`
	ChatID    int64     `json:"chat_id"`
	Reason    string    `json:"reason,omitempty"`
	WarnedBy  int64     `json:"warned_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Cleared   bool      `json:"cleared"`
}

type AdminBanRequest struct {
	ChatID      int64  `json:"chat_id" binding:"required"`
	UserID      int64  `json:"user_id" binding:"required"`
	Reason      string `json:"reason,omitempty"`
	BannedBy    int64  `json:"banned_by,omitempty"`
	OwnerUserID string `json:"-"`
}

type BatchUserRequest struct {
	ChatID      int64   `json:"chat_id" binding:"required"`
	UserIDs     []int64 `json:"user_ids" binding:"required"`
	Action      string  `json:"action" binding:"required"`
	Delta       int     `json:"delta,omitempty"`
	Reason      string  `json:"reason,omitempty"`
	OwnerUserID string  `json:"-"`
}

type BatchUserResult struct {
	SuccessCount int      `json:"success_count"`
	Failed       []string `json:"failed"`
}

type ExportUserQuery struct {
	ChatID      int64  `form:"chatID"`
	Keyword     string `form:"keyword"`
	Status      string `form:"status"`
	OwnerUserID string `form:"-"`
}

type ExportUserRow struct {
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	ChatID      int64     `json:"chat_id"`
	TotalPoints int64     `json:"total_points"`
	Level       string    `json:"level"`
	Status      string    `json:"status"`
	WarnCount   int       `json:"warn_count"`
	JoinedAt    time.Time `json:"joined_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type CommonListQuery struct {
	Limit       int    `form:"limit,default=20" binding:"gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"gte=0"`
	Cursor      string `form:"cursor"`
	OwnerUserID string `form:"-"`
}

type PostCreateRequest struct {
	ChatID            int64      `json:"chat_id" binding:"required"`
	Title             string     `json:"title,omitempty"`
	Content           string     `json:"content,omitempty"`
	MediaURL          string     `json:"media_url,omitempty"`
	MediaName         string     `json:"media_name,omitempty"`
	MediaMime         string     `json:"media_mime,omitempty"`
	MediaDataBase64   string     `json:"media_data_base64,omitempty"`
	MediaType         string     `json:"media_type,omitempty"`
	CronExpr          string     `json:"cron_expr,omitempty"`
	RunOnceAt         *time.Time `json:"run_once_at,omitempty"`
	Enabled           *bool      `json:"enabled,omitempty"`
	PublishAt         *time.Time `json:"publish_at,omitempty"`
	Language          string     `json:"language,omitempty"`
	CreatedBy         string     `json:"created_by,omitempty"`
	TemplateKey       string     `json:"template_key,omitempty"`
	ExternalRef       string     `json:"external_ref,omitempty"`
	PinAfterSend      *bool      `json:"pin_after_send,omitempty"`
	AutoDeleteSeconds int        `json:"auto_delete_seconds,omitempty" binding:"omitempty,gte=0"`
}

type PostUpdateRequest struct {
	Title             *string    `json:"title,omitempty"`
	Content           *string    `json:"content,omitempty"`
	MediaURL          *string    `json:"media_url,omitempty"`
	MediaName         *string    `json:"media_name,omitempty"`
	MediaMime         *string    `json:"media_mime,omitempty"`
	MediaDataBase64   *string    `json:"media_data_base64,omitempty"`
	ClearInlineMedia  *bool      `json:"clear_inline_media,omitempty"`
	MediaType         *string    `json:"media_type,omitempty"`
	CronExpr          *string    `json:"cron_expr,omitempty"`
	RunOnceAt         *time.Time `json:"run_once_at,omitempty"`
	Enabled           *bool      `json:"enabled,omitempty"`
	PublishAt         *time.Time `json:"publish_at,omitempty"`
	Language          *string    `json:"language,omitempty"`
	TemplateKey       *string    `json:"template_key,omitempty"`
	ExternalRef       *string    `json:"external_ref,omitempty"`
	Status            *string    `json:"status,omitempty"`
	PinAfterSend      *bool      `json:"pin_after_send,omitempty"`
	AutoDeleteSeconds *int       `json:"auto_delete_seconds,omitempty" binding:"omitempty,gte=0"`
}

type Post struct {
	ID                string     `json:"id"`
	ChatID            int64      `json:"chat_id"`
	Title             string     `json:"title"`
	Content           string     `json:"content"`
	MediaURL          string     `json:"media_url,omitempty"`
	MediaName         string     `json:"media_name,omitempty"`
	MediaMime         string     `json:"media_mime,omitempty"`
	HasInlineMedia    bool       `json:"has_inline_media,omitempty"`
	MediaType         string     `json:"media_type,omitempty"`
	CronExpr          string     `json:"cron_expr,omitempty"`
	RunOnceAt         *time.Time `json:"run_once_at,omitempty"`
	Enabled           bool       `json:"enabled"`
	LastRunAt         *time.Time `json:"last_run_at,omitempty"`
	PublishAt         *time.Time `json:"publish_at,omitempty"`
	Status            string     `json:"status"`
	Language          string     `json:"language,omitempty"`
	TemplateKey       string     `json:"template_key,omitempty"`
	ExternalRef       string     `json:"external_ref,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	PinAfterSend      bool       `json:"pin_after_send"`
	AutoDeleteSeconds int        `json:"auto_delete_seconds"`
}

type ScheduleCreateRequest struct {
	PostID      string    `json:"post_id,omitempty"`
	ChatID      int64     `json:"chat_id" binding:"required"`
	RunAt       time.Time `json:"run_at" binding:"required"`
	CronExpr    string    `json:"cron_expr,omitempty"`
	Timezone    string    `json:"timezone,omitempty"`
	Enabled     bool      `json:"enabled"`
	RepeatCount *int      `json:"repeat_count,omitempty"`
	Note        string    `json:"note,omitempty"`
}

type ScheduleUpdateRequest struct {
	RunAt       *time.Time `json:"run_at,omitempty"`
	CronExpr    *string    `json:"cron_expr,omitempty"`
	Timezone    *string    `json:"timezone,omitempty"`
	Enabled     *bool      `json:"enabled,omitempty"`
	RepeatCount *int       `json:"repeat_count,omitempty"`
	Note        *string    `json:"note,omitempty"`
	Status      *string    `json:"status,omitempty"`
}

type Schedule struct {
	ID          string    `json:"id"`
	PostID      string    `json:"post_id,omitempty"`
	ChatID      int64     `json:"chat_id"`
	RunAt       time.Time `json:"run_at"`
	CronExpr    string    `json:"cron_expr,omitempty"`
	Timezone    string    `json:"timezone,omitempty"`
	Enabled     bool      `json:"enabled"`
	RepeatCount *int      `json:"repeat_count,omitempty"`
	Note        string    `json:"note,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StatsQuery struct {
	ChatID      int64     `form:"chat_id"`
	From        time.Time `form:"from" time_format:"2006-01-02"`
	To          time.Time `form:"to" time_format:"2006-01-02"`
	OwnerUserID string    `form:"-"`
}

type StatsOverview struct {
	TotalChats     int64 `json:"total_chats"`
	TotalPosts     int64 `json:"total_posts"`
	TotalSchedules int64 `json:"total_schedules"`
	TotalMembers   int64 `json:"total_members"`
	ActiveUsers    int64 `json:"active_users"`
	PointsIssued   int64 `json:"points_issued"`
	OpenTasks      int64 `json:"open_tasks"`
}

type ActivityStats struct {
	Date         string `json:"date"`
	Messages     int64  `json:"messages"`
	Commands     int64  `json:"commands"`
	NewMembers   int64  `json:"new_members"`
	LeavingUsers int64  `json:"leaving_users"`
}

type PointsStats struct {
	Rank     int64  `json:"rank"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username,omitempty"`
	Points   int64  `json:"points"`
	Nickname string `json:"nickname,omitempty"`
}

type LotteryListQuery struct {
	ChatID      int64  `form:"chat_id"`
	Status      string `form:"status"`
	Limit       int    `form:"limit,default=20" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,gte=0"`
	Cursor      string `form:"cursor"`
	OwnerUserID string `form:"-"`
}

type LotteryCreateRequest struct {
	ChatID          int64      `json:"chat_id" binding:"required"`
	Title           string     `json:"title" binding:"required,max=128"`
	Prize           string     `json:"prize"`
	CostPoints      int        `json:"cost_points" binding:"gte=0"`
	MaxParticipants int        `json:"max_participants" binding:"gte=0"`
	WinnerCount     int        `json:"winner_count" binding:"gte=1"`
	EndAt           *time.Time `json:"end_at,omitempty"`
	CreatedBy       int64      `json:"created_by,omitempty"`
	JoinType        string     `json:"join_type,omitempty" binding:"omitempty,oneof=button keyword both"`
	JoinKeyword     string     `json:"join_keyword,omitempty" binding:"omitempty,max=64"`
}

type Lottery struct {
	ID              int64      `json:"id"`
	ChatID          int64      `json:"chat_id"`
	Title           string     `json:"title"`
	Prize           string     `json:"prize"`
	CostPoints      int        `json:"cost_points"`
	MaxParticipants int        `json:"max_participants"`
	WinnerCount     int        `json:"winner_count"`
	EndAt           *time.Time `json:"end_at,omitempty"`
	Status          string     `json:"status"`
	JoinType        string     `json:"join_type"`
	JoinKeyword     string     `json:"join_keyword,omitempty"`
	CreatedBy       int64      `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	EntryCount      int64      `json:"entry_count,omitempty"`
	WinnerCountDone int64      `json:"winner_count_done,omitempty"`
}

type LotteryEntry struct {
	ID        int64     `json:"id"`
	LotteryID int64     `json:"lottery_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	JoinedAt  time.Time `json:"joined_at"`
	IsWinner  bool      `json:"is_winner"`
}

type LevelListQuery struct {
	ChatID      int64  `form:"chat_id"`
	Limit       int    `form:"limit,default=20" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,gte=0"`
	Cursor      string `form:"cursor"`
	OwnerUserID string `form:"-"`
}

type LevelCreateRequest struct {
	ChatID      int64    `json:"chat_id" binding:"required"`
	Level       int      `json:"level,omitempty" binding:"omitempty,gte=1"`
	Name        string   `json:"name" binding:"required,max=64"`
	MinPoints   int64    `json:"min_points" binding:"gte=0"`
	Badge       string   `json:"badge,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type LevelUpdateRequest struct {
	ChatID      int64     `json:"chat_id,omitempty"`
	Level       *int      `json:"level,omitempty" binding:"omitempty,gte=1"`
	Name        *string   `json:"name,omitempty" binding:"omitempty,max=64"`
	MinPoints   *int64    `json:"min_points,omitempty" binding:"omitempty,gte=0"`
	Badge       *string   `json:"badge,omitempty"`
	Permissions *[]string `json:"permissions,omitempty"`
}

type Level struct {
	ID          string    `json:"id"`
	ChatID      int64     `json:"chat_id"`
	Level       int       `json:"level"`
	Name        string    `json:"name"`
	MinPoints   int64     `json:"min_points"`
	Badge       string    `json:"badge,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AdminViolationListQuery struct {
	ChatID      int64  `form:"chat_id"`
	UserID      int64  `form:"user_id"`
	Type        string `form:"type"`
	Status      string `form:"status"`
	Limit       int    `form:"limit,default=20" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,gte=0"`
	Cursor      string `form:"cursor"`
	OwnerUserID string `form:"-"`
}

type AdminViolationUpdateRequest struct {
	Status     *string `json:"status,omitempty"`
	Resolution *string `json:"resolution,omitempty"`
}

type AdminViolation struct {
	ID         string     `json:"id"`
	ChatID     int64      `json:"chat_id"`
	UserID     int64      `json:"user_id"`
	Username   string     `json:"username,omitempty"`
	Type       string     `json:"type"`
	Reason     string     `json:"reason,omitempty"`
	Source     string     `json:"source,omitempty"`
	Status     string     `json:"status"`
	Count      int        `json:"count,omitempty"`
	Resolution string     `json:"resolution,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type KeywordListQuery struct {
	ChatID      int64  `form:"chat_id"`
	Scope       string `form:"scope"`
	Action      string `form:"action"`
	Enabled     *bool  `form:"enabled"`
	Limit       int    `form:"limit,default=20" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,gte=0"`
	OwnerUserID string `form:"-"`
}

type KeywordCreateRequest struct {
	ChatID    int64  `json:"chat_id" binding:"required"`
	Pattern   string `json:"pattern,omitempty" binding:"omitempty,max=128"`
	Keyword   string `json:"keyword,omitempty" binding:"omitempty,max=128"`
	MatchType string `json:"match_type,omitempty"`
	Action    string `json:"action" binding:"required,max=32"`
	Scope     string `json:"scope,omitempty"`
	ReplyText string `json:"reply_text,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

type KeywordUpdateRequest struct {
	Pattern   *string `json:"pattern,omitempty" binding:"omitempty,max=128"`
	Keyword   *string `json:"keyword,omitempty" binding:"omitempty,max=128"`
	MatchType *string `json:"match_type,omitempty"`
	Action    *string `json:"action,omitempty" binding:"omitempty,max=32"`
	Scope     *string `json:"scope,omitempty"`
	ReplyText *string `json:"reply_text,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

type Keyword struct {
	ID        string    `json:"id"`
	ChatID    int64     `json:"chat_id"`
	Pattern   string    `json:"pattern"`
	Keyword   string    `json:"keyword,omitempty"`
	MatchType string    `json:"match_type"`
	Action    string    `json:"action"`
	Scope     string    `json:"scope,omitempty"`
	ReplyText string    `json:"reply_text,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AutoReplyListQuery struct {
	ChatID      int64  `form:"chat_id"`
	Enabled     *bool  `form:"enabled"`
	Limit       int    `form:"limit,default=20" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,gte=0"`
	OwnerUserID string `form:"-"`
}

type AutoReplyCreateRequest struct {
	ChatID    int64  `json:"chat_id" binding:"required"`
	Keyword   string `json:"keyword" binding:"required,max=128"`
	MatchType string `json:"match_type,omitempty"`
	ReplyText string `json:"reply_text" binding:"required"`
	Enabled   *bool  `json:"enabled,omitempty"`
	CreatedBy int64  `json:"created_by,omitempty"`
}

type AutoReplyUpdateRequest struct {
	Keyword   *string `json:"keyword,omitempty" binding:"omitempty,max=128"`
	MatchType *string `json:"match_type,omitempty"`
	ReplyText *string `json:"reply_text,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

type AutoReply struct {
	ID        string    `json:"id"`
	ChatID    int64     `json:"chat_id"`
	Keyword   string    `json:"keyword"`
	MatchType string    `json:"match_type"`
	ReplyText string    `json:"reply_text"`
	Enabled   bool      `json:"enabled"`
	CreatedBy int64     `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BackupImportResponse struct {
	Message string `json:"message"`
}

type BackupData struct {
	Version    string                     `json:"version"`
	ExportedAt string                     `json:"exported_at"`
	Scope      string                     `json:"scope"`
	Tables     map[string]json.RawMessage `json:"tables"`
}

type TemplateListQuery struct {
	ChatID      int64  `form:"chatID"`
	Limit       int    `form:"limit,default=20" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,gte=0"`
	Cursor      string `form:"cursor"`
	OwnerUserID string `form:"-"`
}

type TemplateCreateRequest struct {
	ChatID    *int64 `json:"chat_id,omitempty"`
	Name      string `json:"name" binding:"required"`
	Content   string `json:"content,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	MediaURL  string `json:"media_url,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
	CreatedBy int64  `json:"created_by,omitempty"`
}

type TemplateUpdateRequest struct {
	ChatID    **int64 `json:"chat_id,omitempty"`
	Name      *string `json:"name,omitempty"`
	Content   *string `json:"content,omitempty"`
	MediaType *string `json:"media_type,omitempty"`
	MediaURL  *string `json:"media_url,omitempty"`
	ParseMode *string `json:"parse_mode,omitempty"`
}

type Template struct {
	ID        string    `json:"id"`
	ChatID    *int64    `json:"chat_id,omitempty"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	MediaType string    `json:"media_type"`
	MediaURL  string    `json:"media_url,omitempty"`
	ParseMode string    `json:"parse_mode,omitempty"`
	CreatedBy int64     `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type InviteLinkListQuery struct {
	ChatID      int64  `form:"chatID"`
	Limit       int    `form:"limit,default=20" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,gte=0"`
	Cursor      string `form:"cursor"`
	OwnerUserID string `form:"-"`
}

type InviteLinkCreateRequest struct {
	ChatID             int64  `json:"chat_id" binding:"required"`
	Name               string `json:"name,omitempty"`
	CreatesJoinRequest bool   `json:"creates_join_request,omitempty"`
	CreatedBy          int64  `json:"created_by,omitempty"`
}

type InviteLink struct {
	ID                 string    `json:"id"`
	ChatID             int64     `json:"chat_id"`
	Name               string    `json:"name"`
	InviteLink         string    `json:"invite_link"`
	CreatesJoinRequest bool      `json:"creates_join_request"`
	JoinCount          int       `json:"join_count"`
	CreatedBy          int64     `json:"created_by,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UserListQuery struct {
	ChatID      int64  `form:"chat_id"`
	Keyword     string `form:"keyword"`
	Limit       int    `form:"limit,default=50" binding:"omitempty,gte=1,lte=100"`
	Offset      int    `form:"offset,default=0" binding:"omitempty,gte=0"`
	OwnerUserID string `form:"-"`
}

type UserRecord struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"display_name"`
	ChatID      int64     `json:"chat_id"`
	TotalPoints int64     `json:"total_points"`
	Status      string    `json:"status"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type AdminAuthService interface {
	Authenticate(ctx context.Context, req AdminLoginRequest) (AdminIdentity, error)
	AuthenticateTelegram(ctx context.Context, req TelegramLoginRequest) (AdminIdentity, error)
}

type BotConfigService interface {
	Get(ctx context.Context) (*BotConfig, error)
	Update(ctx context.Context, req BotConfigUpdateRequest) (*BotConfig, error)
}

type ChatBindingService interface {
	Bind(ctx context.Context, req ChatBindingRequest) (*ChatBinding, error)
	Unbind(ctx context.Context, chatID int64) error
	List(ctx context.Context, query CommonListQuery) ([]ChatBinding, error)
	UserOwnsChat(ctx context.Context, userID string, chatID string) (bool, error)
}

type ChatPointConfigService interface {
	Get(ctx context.Context, chatID int64) (*ChatPointConfig, error)
	Update(ctx context.Context, chatID int64, req ChatPointConfigUpdateRequest) (*ChatPointConfig, error)
}

type PointsAdminService interface {
	GetRank(ctx context.Context, chatID int64, period string, limit int) ([]PointRankItem, error)
	GetUser(ctx context.Context, chatID int64, userID int64) (*PointUserResponse, error)
	AdjustUser(ctx context.Context, chatID int64, userID int64, delta int, reason string) (*PointUserResponse, error)
	ListLogs(ctx context.Context, chatID int64, userID int64, query PointLogListQuery) (*PointLogListResponse, error)
}

type ChatAdminService interface {
	GetConfig(ctx context.Context, chatID int64) (*ChatAdminConfig, error)
	UpdateConfig(ctx context.Context, chatID int64, req ChatAdminConfigUpdateRequest) (*ChatAdminConfig, error)
	ListBans(ctx context.Context, chatID int64, query CommonListQuery) ([]BanLog, error)
	Ban(ctx context.Context, req AdminBanRequest) error
	Unban(ctx context.Context, chatID int64, userID int64) error
	ListWarns(ctx context.Context, chatID int64, userID int64) ([]WarnRecord, error)
	ExportUserRows(ctx context.Context, query ExportUserQuery) ([]ExportUserRow, error)
	BatchUserAction(ctx context.Context, req BatchUserRequest) (*BatchUserResult, error)
}

type LotteryService interface {
	List(ctx context.Context, query LotteryListQuery) ([]Lottery, error)
	Create(ctx context.Context, req LotteryCreateRequest) (*Lottery, error)
	Cancel(ctx context.Context, id int64, ownerUserID string) error
	Entries(ctx context.Context, id int64, ownerUserID string) ([]LotteryEntry, error)
	Winners(ctx context.Context, id int64, ownerUserID string) ([]LotteryEntry, error)
}

type LevelService interface {
	List(ctx context.Context, query LevelListQuery) ([]Level, error)
	Create(ctx context.Context, req LevelCreateRequest) (*Level, error)
	Update(ctx context.Context, id string, req LevelUpdateRequest, ownerUserID string) (*Level, error)
	Delete(ctx context.Context, id string, ownerUserID string) error
}

type AdminViolationService interface {
	List(ctx context.Context, query AdminViolationListQuery) (*CursorListResponse[AdminViolation], error)
	Update(ctx context.Context, id string, req AdminViolationUpdateRequest, ownerUserID string) (*AdminViolation, error)
}

type KeywordService interface {
	List(ctx context.Context, query KeywordListQuery) ([]Keyword, error)
	Create(ctx context.Context, req KeywordCreateRequest) (*Keyword, error)
	Update(ctx context.Context, id string, req KeywordUpdateRequest, ownerUserID string) (*Keyword, error)
	Delete(ctx context.Context, id string, ownerUserID string) error
}

type AutoReplyService interface {
	List(ctx context.Context, query AutoReplyListQuery) ([]AutoReply, error)
	Create(ctx context.Context, req AutoReplyCreateRequest) (*AutoReply, error)
	Update(ctx context.Context, id string, req AutoReplyUpdateRequest, ownerUserID string) (*AutoReply, error)
	Delete(ctx context.Context, id string, ownerUserID string) error
}

type BackupService interface {
	Export(ctx context.Context, scope string) (*BackupData, error)
	Import(ctx context.Context, data *BackupData, mode string) error
}

type TemplateService interface {
	List(ctx context.Context, query TemplateListQuery) (*CursorListResponse[Template], error)
	Create(ctx context.Context, req TemplateCreateRequest, ownerUserID string) (*Template, error)
	Update(ctx context.Context, id string, req TemplateUpdateRequest, ownerUserID string) (*Template, error)
	Delete(ctx context.Context, id string, ownerUserID string) error
}

type InviteLinkService interface {
	List(ctx context.Context, query InviteLinkListQuery) (*CursorListResponse[InviteLink], error)
	Create(ctx context.Context, req InviteLinkCreateRequest) (*InviteLink, error)
	Delete(ctx context.Context, id string, ownerUserID string) error
}

type PostService interface {
	Create(ctx context.Context, req PostCreateRequest) (*Post, error)
	List(ctx context.Context, query CommonListQuery) ([]Post, error)
	Get(ctx context.Context, id string, ownerUserID string) (*Post, error)
	Update(ctx context.Context, id string, req PostUpdateRequest, ownerUserID string) (*Post, error)
	Delete(ctx context.Context, id string, ownerUserID string) error
	Toggle(ctx context.Context, id string, ownerUserID string) (*Post, error)
}

type ScheduleService interface {
	Create(ctx context.Context, req ScheduleCreateRequest) (*Schedule, error)
	List(ctx context.Context, query CommonListQuery) ([]Schedule, error)
	Get(ctx context.Context, id string, ownerUserID string) (*Schedule, error)
	Update(ctx context.Context, id string, req ScheduleUpdateRequest, ownerUserID string) (*Schedule, error)
	Delete(ctx context.Context, id string, ownerUserID string) error
}

type StatsService interface {
	Overview(ctx context.Context, query StatsQuery) (*StatsOverview, error)
	Activity(ctx context.Context, query StatsQuery) ([]ActivityStats, error)
	Points(ctx context.Context, query StatsQuery) ([]PointsStats, error)
}

type UserService interface {
	List(ctx context.Context, query UserListQuery) ([]UserRecord, error)
}
