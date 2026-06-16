package bot

import (
	"context"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	PermissionAdmin    = "admin"
	PermissionModerate = "moderate"
	PermissionPublish  = "publish"
	PermissionStats    = "stats"
	PermissionVerify   = "verify"
	PermissionKeyword  = "keyword"
	PermissionPoints   = "points"

	CallbackPrefix = "op"
)

type HandlerFunc = handlers.Response

type Options struct {
	DefaultLocale         string
	MiniAppURL            string
	TurnstileVerifySecret string
	Features              Features
}

// Features controls which functional modules are registered.
// The zero value enables everything; add names to Disabled to turn modules off.
type Features struct {
	disabled map[string]bool
}

// NewFeatures returns a Features that disables the named modules.
// Passing an empty slice enables everything (the default).
func NewFeatures(disabled []string) Features {
	if len(disabled) == 0 {
		return Features{}
	}
	m := make(map[string]bool, len(disabled))
	for _, n := range disabled {
		m[n] = true
	}
	return Features{disabled: m}
}

// Enabled reports whether the named module should be registered.
// Any name not in the disabled set (including with a nil map) returns true.
func (f Features) Enabled(name string) bool {
	return !f.disabled[name]
}

type App struct {
	services   Services
	options    Options
	miniAppURL string
	router     *CallbackRouter
	state      *memoryStateStore
}

type Services struct {
	Access         AccessService
	TelegramAccess TelegramAccessService
	ChatBindings   ChatBindingService
	RateLimit      RateLimitService
	Points         PointsService
	Admin          AdminService
	Lottery        LotteryService
	Publish        PublishService
	Level          LevelService
	KeywordFilter  KeywordFilterService
	AutoReply      AutoReplyService
	InviteLink     InviteLinkService
	Templates      MessageTemplateService
	Violations     ViolationService
	AuditLog       AuditLogService
	AiFilter       AiFilterService
	Redis          RedisStateService
}

type Actor struct {
	ID           int64
	Username     string
	FirstName    string
	IsBot        bool
	LanguageCode string
}

type ChatRef struct {
	ID       int64
	Type     string
	Title    string
	Username string
}

type RequestScope struct {
	Context      context.Context
	Actor        Actor
	Chat         ChatRef
	CallbackData string
}

type AccessService interface {
	IsAdmin(ctx context.Context, chatID int64, userID int64) (bool, error)
	HasPermission(ctx context.Context, chatID int64, userID int64, permission string) (bool, error)
}

type BotAdminStatus struct {
	ChatID                  int64
	BotID                   int64
	Status                  string
	IsAdmin                 bool
	CanManageChat           bool
	CanPostMessages         bool
	CanEditMessages         bool
	CanDeleteMessages       bool
	CanRestrictMembers      bool
	CanInviteUsers          bool
	CanPinMessages          bool
	CanPromoteMembers       bool
	CanManageTopics         bool
	CanManageDirectMessages bool
}

type TelegramAccessService interface {
	CheckBotAdmin(ctx context.Context, tgBot *gotgbot.Bot, chatID int64) (BotAdminStatus, error)
	CheckUserAdmin(ctx context.Context, tgBot *gotgbot.Bot, chatID int64, userID int64) (BotAdminStatus, error)
}

type ChatBindingService interface {
	Bind(ctx context.Context, req api.ChatBindingRequest) (*api.ChatBinding, error)
	List(ctx context.Context, query api.CommonListQuery) ([]api.ChatBinding, error)
	ListByTelegramUser(ctx context.Context, telegramUserID int64, limit int) ([]api.ChatBinding, error)
}

type RedisStateService interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type RateLimitService interface {
	Allow(ctx context.Context, key string, cost int) (allowed bool, retryAfter time.Duration, err error)
}

type PointsService interface {
	GetSummary(ctx context.Context, chatID int64, userID int64) (string, error)
	GetRank(ctx context.Context, chatID int64, period string, limit int) (string, error)
	Adjust(ctx context.Context, chatID int64, userID int64, delta int, reason string) error
	GetActivityStats(ctx context.Context, chatID int64, window string) (string, error)
	GetConfig(ctx context.Context, chatID int64) (ChatPointConfig, error)
	UpdateConfig(ctx context.Context, chatID int64, patch ChatPointConfigPatch) (ChatPointConfig, error)
	ToggleConfig(ctx context.Context, chatID int64) (ChatPointConfig, error)
	AwardMessage(ctx context.Context, req PointAwardRequest) (PointAwardResult, error)
}

type ChatPointConfig struct {
	ChatID          int64
	PointText       int
	PointPhoto      int
	PointSticker    int
	PointVideo      int
	PointFile       int
	PointVoice      int
	CooldownSeconds int
	Enabled         bool
}

type ChatPointConfigPatch struct {
	PointText       *int
	PointPhoto      *int
	PointSticker    *int
	PointVideo      *int
	PointFile       *int
	PointVoice      *int
	CooldownSeconds *int
	Enabled         *bool
}

type PointAwardRequest struct {
	ChatID        int64
	UserID        int64
	MessageID     int64
	MessageType   string
	CooldownScope string
	ReasonPrefix  string
	Username      string
	DisplayName   string
	ChatType      string
	ChatTitle     string
	IsForwarded   bool
	IsCommand     bool
	FromBot       bool
}

type PointAwardResult struct {
	Awarded bool
	Points  int
	Reason  string
}

type ChatAdminConfig struct {
	ChatID             int64
	WelcomeText        string
	VerifyEnabled      bool
	VerifyType         string
	VerifyTimeout      int
	WarnLimit          int
	VerifyQuestion     string
	VerifyOptions      string
	VerifyCorrectIndex int
	VerifyWhitelist    string
	VerifyDifficulty   string
	RulesText          string
}

type ChatAdminConfigPatch struct {
	WelcomeText        *string
	VerifyEnabled      *bool
	VerifyType         *string
	VerifyTimeout      *int
	WarnLimit          *int
	VerifyQuestion     *string
	VerifyOptions      *string
	VerifyCorrectIndex *int
	VerifyWhitelist    *string
	VerifyDifficulty   *string
	RulesText          *string
}

type BanLog struct {
	ID         uint64
	UserID     int64
	ChatID     int64
	Reason     string
	BannedBy   int64
	BannedAt   time.Time
	UnbannedAt *time.Time
}

type WarnRecord struct {
	ID        uint64
	UserID    int64
	ChatID    int64
	Reason    string
	WarnedBy  int64
	CreatedAt time.Time
	Cleared   bool
}

type AdminService interface {
	GetConfig(ctx context.Context, chatID int64) (ChatAdminConfig, error)
	UpdateConfig(ctx context.Context, chatID int64, patch ChatAdminConfigPatch) (ChatAdminConfig, error)
	ToggleVerify(ctx context.Context, chatID int64) (ChatAdminConfig, error)
	RecordBan(ctx context.Context, chatID int64, userID int64, operatorID int64, reason string) error
	RecordUnban(ctx context.Context, chatID int64, userID int64, operatorID int64) error
	RecordWarn(ctx context.Context, chatID int64, userID int64, operatorID int64, reason string) (int64, int, error)
	ClearWarns(ctx context.Context, chatID int64, userID int64) error
	CountWarns(ctx context.Context, chatID int64, userID int64) (int64, error)
	ListWarns(ctx context.Context, chatID int64, userID int64) ([]WarnRecord, error)
	ListBans(ctx context.Context, chatID int64, limit int) ([]BanLog, error)
	SetVerifyChallenge(ctx context.Context, chatID int64, userID int64, challenge VerifyChallenge, ttl time.Duration) error
	CheckVerifyChallenge(ctx context.Context, chatID int64, userID int64, answer string) (VerifyCheckResult, error)
	GetVerifyChallenge(ctx context.Context, chatID int64, userID int64) (VerifyChallenge, bool, error)
	ClearVerifyChallenge(ctx context.Context, chatID int64, userID int64) error
	RecordVerifyEvent(ctx context.Context, chatID int64, userID int64, eventType string, detail string) error
	GetVerifyStats(ctx context.Context, chatID int64) (VerifyStats, error)
	RecordSeenUser(ctx context.Context, chatID int64, userID int64) error
	ListSeenUsers(ctx context.Context, chatID int64) ([]int64, error)
}

type VerifyChallenge struct {
	Answer     string
	MessageID  int64
	Attempts   int
	ExpireAt   time.Time
	Question   string
	MemberName string
	PollID     string
}

type VerifyCheckResult struct {
	OK                bool
	Expired           bool
	RemainingAttempts int
	ShouldKick        bool
	Challenge         VerifyChallenge
}

type VerifyStats struct {
	TotalPassed  int64
	TotalFailed  int64
	TotalTimeout int64
	PendingCount int64
}

type LotteryService interface {
	Create(ctx context.Context, req api.LotteryCreateRequest) (*api.Lottery, error)
	GetItem(ctx context.Context, chatID int64, lotteryID int64) (api.Lottery, error)
	ListItems(ctx context.Context, chatID int64, limit int) ([]api.Lottery, error)
	ListActiveItems(ctx context.Context, chatID int64, limit int) ([]api.Lottery, error)
	ListActive(ctx context.Context, chatID int64) (string, error)
	Info(ctx context.Context, chatID int64, lotteryID int64) (string, error)
	Join(ctx context.Context, chatID int64, lotteryID int64, userID int64, username ...string) (string, error)
	JoinByKeyword(ctx context.Context, chatID int64, keyword string, userID int64, username string) (bool, string, int64, error)
	CancelForChat(ctx context.Context, chatID int64, lotteryID int64, operatorID int64) (string, error)
}

type PublishService interface {
	PreviewQueue(ctx context.Context, chatID int64) (string, error)
	ListScheduledPosts(ctx context.Context, chatID int64, limit int) (string, error)
	ListScheduledPostItems(ctx context.Context, chatID int64, limit int) ([]ScheduledPostItem, error)
	CreateScheduledPost(ctx context.Context, req ScheduledPostCreate) (ScheduledPostItem, error)
	ToggleScheduledPost(ctx context.Context, chatID int64, postID uint64) (ScheduledPostItem, error)
	DeleteScheduledPost(ctx context.Context, chatID int64, postID uint64) error
	RecordQuickPost(ctx context.Context, chatID int64, content string, operatorID int64) error
	SyncChannel(ctx context.Context, chatID int64, operatorID int64) error
}

type ScheduledPostCreate struct {
	ChatID            int64
	Title             string
	Content           string
	MediaURL          string
	MediaType         string
	CronExpr          string
	RunOnceAt         *time.Time
	Enabled           bool
	PinAfterSend      bool
	AutoDeleteSeconds int
	CreatedBy         int64
}

type ScheduledPostItem struct {
	ID                uint64
	ChatID            int64
	Title             string
	Content           string
	MediaURL          string
	MediaType         string
	CronExpr          string
	RunOnceAt         *time.Time
	Enabled           bool
	LastRunAt         *time.Time
	CreatedAt         time.Time
	PinAfterSend      bool
	AutoDeleteSeconds int
}

type LevelService interface {
	SetLevel(ctx context.Context, chatID int64, userID int64, level int, operatorID int64) (string, error)
	ListLevelRules(ctx context.Context, chatID int64) (string, error)
	UpsertLevelRule(ctx context.Context, chatID int64, level int, name string, minPoints int64, badge string) (string, error)
	DeleteLevelRule(ctx context.Context, chatID int64, level int) (string, error)
}

type KeywordFilterService interface {
	ListKeywords(ctx context.Context, chatID int64) (string, error)
	AddKeyword(ctx context.Context, chatID int64, keyword string, operatorID int64) (string, error)
	DeleteKeyword(ctx context.Context, chatID int64, keyword string, operatorID int64) (string, error)
	GetModerationConfig(ctx context.Context, chatID int64) (ChatModerationConfig, error)
	MatchKeyword(ctx context.Context, chatID int64, text string) (KeywordFilterMatch, error)
	RecordKeywordViolation(ctx context.Context, violation KeywordViolation) error
}

type ChatModerationConfig struct {
	ChatID               int64
	BlockLinks           bool
	LinkWhitelist        []string
	LinkBlacklist        []string
	BlockForwards        bool
	BlockMedia           bool
	KeywordFilterEnabled bool
	SpamScoreThreshold   int
	AiFilterEnabled      bool
	RestrictUnverified   bool
}

type KeywordFilterMatch struct {
	Matched   bool
	Keyword   string
	MatchType string
	Action    string
	ReplyText string
}

type KeywordViolation struct {
	UserID          int64
	ChatID          int64
	ViolationType   string
	ActionTaken     string
	MessageText     string
	DetectedBy      string
	DurationSeconds int
}

type AutoReplyMatch struct {
	Keyword   string
	ReplyText string
}

type AutoReplyCreate struct {
	ChatID    int64
	Keyword   string
	MatchType string
	ReplyText string
	Enabled   bool
	CreatedBy int64
}

type AutoReplyRecord struct {
	ID        string
	ChatID    int64
	Keyword   string
	MatchType string
	ReplyText string
	Enabled   bool
	CreatedBy int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AutoReplyService interface {
	MatchAll(ctx context.Context, chatID int64, text string) ([]AutoReplyMatch, error)
	ListForChat(ctx context.Context, chatID int64) ([]AutoReplyRecord, error)
	CreateForBot(ctx context.Context, req AutoReplyCreate) (AutoReplyRecord, error)
	DeleteByKeyword(ctx context.Context, chatID int64, keyword string) error
}

type InviteLinkService interface {
	IncrementJoinCount(ctx context.Context, chatID int64, inviteLink string) error
	ListForChat(ctx context.Context, chatID int64, limit int) (string, error)
	CreateForBot(ctx context.Context, chatID int64, name string, createsJoinRequest bool, operatorID int64) (InviteLinkRecord, error)
	DeleteForChat(ctx context.Context, chatID int64, id string) error
}

type MessageTemplateCreate struct {
	ChatID    int64
	Name      string
	Content   string
	MediaType string
	MediaURL  string
	ParseMode string
	CreatedBy int64
}

type MessageTemplateRecord struct {
	ID        string
	ChatID    *int64
	Name      string
	Content   string
	MediaType string
	MediaURL  string
	ParseMode string
	CreatedBy int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MessageTemplateService interface {
	ListForChat(ctx context.Context, chatID int64, limit int) ([]MessageTemplateRecord, error)
	CreateForBot(ctx context.Context, req MessageTemplateCreate) (MessageTemplateRecord, error)
	DeleteForChat(ctx context.Context, chatID int64, id string) error
}

type AuditLogService interface {
	Log(entry AuditLogEntry)
}

// AiFilterService uses an LLM to judge whether a message is spam.
type AiFilterService interface {
	IsSpam(ctx context.Context, text string, userName string) (bool, string, error)
}

type AuditLogEntry struct {
	ActorTelegramID  int64
	ChatTelegramID   int64
	Action           string
	EntityType       string
	TargetTelegramID int64
	Detail           string
}

type ViolationService interface {
	ListViolations(ctx context.Context, chatID int64, userID int64, limit int, offset int) ([]model.ViolationRecord, error)
	UpdateViolation(ctx context.Context, id string, status *string, resolution *string) (model.ViolationRecord, error)
}

type InviteLinkRecord struct {
	ID                 string
	ChatID             int64
	Name               string
	InviteLink         string
	CreatesJoinRequest bool
	JoinCount          int
	CreatedBy          int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
