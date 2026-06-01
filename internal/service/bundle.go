package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

type Bundle struct {
	store           *store.Store
	Access          bot.AccessService
	TelegramAccess  bot.TelegramAccessService
	RateLimit       bot.RateLimitService
	Points          bot.PointsService
	Admin           bot.AdminService
	Lottery         bot.LotteryService
	Publish         bot.PublishService
	Levels          *LevelService
	Moderation      *ModerationService
	AutoReply       *AutoReplyService
	Backup          *BackupService
	MessageTemplate *MessageTemplateService
	InviteLink      *InviteLinkService
}

func NewBundle(st *store.Store, redisClient *redis.Client) *Bundle {
	return NewBundleWithBotToken(st, redisClient, "")
}

func NewBundleWithBotToken(st *store.Store, redisClient *redis.Client, botToken string) *Bundle {
	return &Bundle{
		store:           st,
		Access:          NewAccessService(st),
		TelegramAccess:  NewTelegramAccessService(),
		RateLimit:       NewRateLimitService(redisClient),
		Points:          NewPointsService(st),
		Admin:           NewAdminService(st, redisClient),
		Lottery:         NewLotteryService(st, redisClient),
		Publish:         NewPublishService(st),
		Levels:          NewLevelService(st),
		Moderation:      NewModerationService(st),
		AutoReply:       NewAutoReplyService(st),
		Backup:          NewBackupService(st),
		MessageTemplate: NewMessageTemplateService(st),
		InviteLink:      NewInviteLinkService(st, botToken),
	}
}

func (b *Bundle) BotServices() bot.Services {
	return bot.Services{
		Access:         b.Access,
		TelegramAccess: b.TelegramAccess,
		ChatBindings:   &chatBindingService{store: b.store},
		RateLimit:      b.RateLimit,
		Points:         b.Points,
		Admin:          b.Admin,
		Lottery:        b.Lottery,
		Publish:        b.Publish,
		Level:          b.Levels,
		KeywordFilter:  b.Moderation,
		AutoReply:      b.AutoReply,
		InviteLink:     b.InviteLink,
		Templates:      b.MessageTemplate,
		Violations:     b.Moderation,
	}
}

type AccessService struct{ store *store.Store }

func NewAccessService(st *store.Store) *AccessService { return &AccessService{store: st} }

func (s *AccessService) IsAdmin(ctx context.Context, chatID int64, userID int64) (bool, error) {
	return s.HasPermission(ctx, chatID, userID, bot.PermissionAdmin)
}

func (s *AccessService) HasPermission(ctx context.Context, chatID int64, userID int64, permission string) (bool, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return true, nil
	}

	var user model.User
	if err := s.store.DB.WithContext(ctx).Where("telegram_user_id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	var chat model.TelegramChat
	if err := s.store.DB.WithContext(ctx).Where("telegram_chat_id = ?", chatID).First(&chat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	var admin model.ChatAdmin
	err := s.store.DB.WithContext(ctx).Where("chat_id = ? AND user_id = ?", chat.ID, user.ID).First(&admin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	if admin.Role == bot.PermissionAdmin || admin.CanManage {
		return true, nil
	}
	switch permission {
	case bot.PermissionPublish:
		return admin.CanPost, nil
	case bot.PermissionModerate:
		return admin.CanBan || admin.CanDelete, nil
	default:
		return admin.CanManage, nil
	}
}

type RateLimitService struct {
	redis  *redis.Client
	limit  int64
	window time.Duration
}

func NewRateLimitService(client *redis.Client) *RateLimitService {
	return &RateLimitService{redis: client, limit: 30, window: time.Minute}
}

func (s *RateLimitService) Allow(ctx context.Context, key string, cost int) (bool, time.Duration, error) {
	if s == nil || s.redis == nil {
		return true, 0, nil
	}
	if cost <= 0 {
		cost = 1
	}
	count, err := s.redis.IncrBy(ctx, key, int64(cost)).Result()
	if err != nil {
		return true, 0, nil
	}
	if count == int64(cost) {
		_ = s.redis.Expire(ctx, key, s.window).Err()
	}
	if count > s.limit {
		ttl, err := s.redis.TTL(ctx, key).Result()
		if err != nil || ttl <= 0 {
			ttl = s.window
		}
		return false, ttl, nil
	}
	return true, 0, nil
}

type PointsService struct{ store *store.Store }

func NewPointsService(st *store.Store) *PointsService { return &PointsService{store: st} }

type PublishService struct{ store *store.Store }

func NewPublishService(st *store.Store) *PublishService { return &PublishService{store: st} }

func (s *PublishService) PreviewQueue(ctx context.Context, chatID int64) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "发布服务尚未接入。", nil
	}
	var count int64
	if err := s.store.DB.WithContext(ctx).Model(&model.ScheduledPost{}).Where("chat_id = ? AND enabled = ?", chatID, true).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("待发送任务数：%d", count), nil
}

func (s *PublishService) ListScheduledPosts(ctx context.Context, chatID int64, limit int) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "发布服务尚未接入。", nil
	}
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	var posts []model.ScheduledPost
	if err := s.store.DB.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Order("created_at desc").
		Limit(limit).
		Find(&posts).Error; err != nil {
		return "", err
	}
	if len(posts) == 0 {
		return "本群暂无定时发帖任务。", nil
	}
	var builder strings.Builder
	builder.WriteString("本群定时发帖任务：")
	for _, post := range posts {
		status := "关闭"
		if post.Enabled {
			status = "启用"
		}
		schedule := "未设置"
		if post.CronExpr != "" {
			schedule = "cron " + post.CronExpr
		} else if post.RunOnceAt != nil {
			schedule = post.RunOnceAt.Format("2006-01-02 15:04")
		}
		title := strings.TrimSpace(post.Title)
		if title == "" {
			title = strings.TrimSpace(post.Content)
		}
		if len([]rune(title)) > 24 {
			title = string([]rune(title)[:24]) + "..."
		}
		builder.WriteString(fmt.Sprintf("\n#%d [%s] %s - %s", post.ID, status, schedule, title))
	}
	return builder.String(), nil
}

func (s *PublishService) ListScheduledPostItems(ctx context.Context, chatID int64, limit int) ([]bot.ScheduledPostItem, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []bot.ScheduledPostItem{}, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 10
	}
	var posts []model.ScheduledPost
	if err := s.store.DB.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Order("created_at desc").
		Limit(limit).
		Find(&posts).Error; err != nil {
		return nil, err
	}
	out := make([]bot.ScheduledPostItem, 0, len(posts))
	for _, post := range posts {
		out = append(out, scheduledPostToBot(post))
	}
	return out, nil
}

func (s *PublishService) CreateScheduledPost(ctx context.Context, req bot.ScheduledPostCreate) (bot.ScheduledPostItem, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return bot.ScheduledPostItem{}, gorm.ErrInvalidDB
	}
	mediaType := strings.ToLower(strings.TrimSpace(req.MediaType))
	switch mediaType {
	case "", "text":
		mediaType = "text"
	case "photo", "video", "document":
	default:
		mediaType = "text"
	}
	if req.ChatID == 0 {
		return bot.ScheduledPostItem{}, fmt.Errorf("chat_id is required")
	}
	if mediaType == "text" && strings.TrimSpace(req.Content) == "" {
		return bot.ScheduledPostItem{}, fmt.Errorf("content is required")
	}
	if mediaType != "text" && strings.TrimSpace(req.MediaURL) == "" {
		return bot.ScheduledPostItem{}, fmt.Errorf("media_url is required")
	}
	if strings.TrimSpace(req.CronExpr) == "" && req.RunOnceAt == nil {
		return bot.ScheduledPostItem{}, fmt.Errorf("run_once_at or cron_expr is required")
	}
	now := time.Now()
	post := model.ScheduledPost{
		ChatID:            req.ChatID,
		Title:             strings.TrimSpace(req.Title),
		Content:           strings.TrimSpace(req.Content),
		MediaURL:          strings.TrimSpace(req.MediaURL),
		MediaType:         mediaType,
		CronExpr:          strings.TrimSpace(req.CronExpr),
		RunOnceAt:         req.RunOnceAt,
		Enabled:           req.Enabled,
		CreatedAt:         now,
		PinAfterSend:      req.PinAfterSend,
		AutoDeleteSeconds: req.AutoDeleteSeconds,
	}
	if post.Title == "" {
		post.Title = "Bot 创建"
	}
	if err := s.store.DB.WithContext(ctx).Select("*").Create(&post).Error; err != nil {
		return bot.ScheduledPostItem{}, err
	}
	if !req.Enabled {
		if err := s.store.DB.WithContext(ctx).Model(&model.ScheduledPost{}).Where("id = ?", post.ID).Update("enabled", false).Error; err != nil {
			return bot.ScheduledPostItem{}, err
		}
		post.Enabled = false
	}
	return scheduledPostToBot(post), nil
}

func (s *PublishService) ToggleScheduledPost(ctx context.Context, chatID int64, postID uint64) (bot.ScheduledPostItem, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return bot.ScheduledPostItem{}, gorm.ErrInvalidDB
	}
	var post model.ScheduledPost
	if err := s.store.DB.WithContext(ctx).Where("id = ? AND chat_id = ?", postID, chatID).First(&post).Error; err != nil {
		return bot.ScheduledPostItem{}, err
	}
	post.Enabled = !post.Enabled
	if err := s.store.DB.WithContext(ctx).Model(&model.ScheduledPost{}).Where("id = ? AND chat_id = ?", postID, chatID).Update("enabled", post.Enabled).Error; err != nil {
		return bot.ScheduledPostItem{}, err
	}
	return scheduledPostToBot(post), nil
}

func (s *PublishService) DeleteScheduledPost(ctx context.Context, chatID int64, postID uint64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	return s.store.DB.WithContext(ctx).Where("id = ? AND chat_id = ?", postID, chatID).Delete(&model.ScheduledPost{}).Error
}

func (s *PublishService) RecordQuickPost(ctx context.Context, chatID int64, content string, operatorID int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	now := time.Now()
	return s.store.DB.WithContext(ctx).Create(&model.ScheduledPost{
		ChatID:    chatID,
		Title:     "快捷发布",
		Content:   strings.TrimSpace(content),
		MediaType: "text",
		Enabled:   false,
		LastRunAt: &now,
		CreatedAt: now,
	}).Error
}

func (s *PublishService) SyncChannel(ctx context.Context, chatID int64, operatorID int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	return s.store.DB.WithContext(ctx).Create(&model.AuditLog{
		Action:       "sync_channel",
		EntityType:   "telegram_chat",
		MetadataJSON: fmt.Sprintf(`{"telegram_chat_id":%d,"operator_id":%d}`, chatID, operatorID),
		OccurredAt:   time.Now(),
	}).Error
}

func scheduledPostToBot(post model.ScheduledPost) bot.ScheduledPostItem {
	return bot.ScheduledPostItem{
		ID:                post.ID,
		ChatID:            post.ChatID,
		Title:             post.Title,
		Content:           post.Content,
		MediaURL:          post.MediaURL,
		MediaType:         post.MediaType,
		CronExpr:          post.CronExpr,
		RunOnceAt:         post.RunOnceAt,
		Enabled:           post.Enabled,
		LastRunAt:         post.LastRunAt,
		CreatedAt:         post.CreatedAt,
		PinAfterSend:      post.PinAfterSend,
		AutoDeleteSeconds: post.AutoDeleteSeconds,
	}
}

func (s *LevelService) ListLevelRules(ctx context.Context, chatID int64) (string, error) {
	rules, err := s.ListRules(ctx, chatID)
	if err != nil {
		return "", err
	}
	if len(rules) == 0 {
		return "本群暂无等级规则。", nil
	}
	var builder strings.Builder
	builder.WriteString("本群等级规则：")
	for _, rule := range rules {
		builder.WriteString(fmt.Sprintf("\nLv.%d %s：%d 分起，徽章 %s", rule.Level, rule.Name, rule.MinPoints, rule.Badge))
	}
	return builder.String(), nil
}

func (s *LevelService) UpsertLevelRule(ctx context.Context, chatID int64, level int, name string, minPoints int64, badge string) (string, error) {
	rule, err := s.UpsertRule(ctx, LevelRule{
		ChatID:    chatID,
		Level:     level,
		Name:      name,
		MinPoints: minPoints,
		Badge:     badge,
		Enabled:   true,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已保存等级规则：Lv.%d %s，%d 分起，徽章 %s", rule.Level, rule.Name, rule.MinPoints, rule.Badge), nil
}

func (s *LevelService) DeleteLevelRule(ctx context.Context, chatID int64, level int) (string, error) {
	if err := s.DeleteRule(ctx, chatID, level); err != nil {
		return "", err
	}
	return fmt.Sprintf("已删除本群 Lv.%d 等级规则。", level), nil
}

func (s *MessageTemplateService) ListForChat(ctx context.Context, chatID int64, limit int) ([]bot.MessageTemplateRecord, error) {
	records, err := s.List(ctx, MessageTemplateListFilter{ChatID: chatID, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]bot.MessageTemplateRecord, 0, len(records))
	for _, record := range records {
		out = append(out, bot.MessageTemplateRecord{
			ID:        record.ID.String(),
			ChatID:    record.ChatID,
			Name:      record.Name,
			Content:   record.Content,
			MediaType: record.MediaType,
			MediaURL:  record.MediaURL,
			ParseMode: record.ParseMode,
			CreatedBy: record.CreatedBy,
			CreatedAt: record.CreatedAt,
			UpdatedAt: record.UpdatedAt,
		})
	}
	return out, nil
}

func (s *MessageTemplateService) CreateForBot(ctx context.Context, req bot.MessageTemplateCreate) (bot.MessageTemplateRecord, error) {
	chatID := req.ChatID
	record, err := s.Create(ctx, MessageTemplateCreate{
		ChatID:    &chatID,
		Name:      req.Name,
		Content:   req.Content,
		MediaType: req.MediaType,
		MediaURL:  req.MediaURL,
		ParseMode: req.ParseMode,
		CreatedBy: req.CreatedBy,
	})
	if err != nil {
		return bot.MessageTemplateRecord{}, err
	}
	return bot.MessageTemplateRecord{
		ID:        record.ID.String(),
		ChatID:    record.ChatID,
		Name:      record.Name,
		Content:   record.Content,
		MediaType: record.MediaType,
		MediaURL:  record.MediaURL,
		ParseMode: record.ParseMode,
		CreatedBy: record.CreatedBy,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

func (s *MessageTemplateService) DeleteForChat(ctx context.Context, chatID int64, id string) error {
	records, err := s.List(ctx, MessageTemplateListFilter{ChatID: chatID, Limit: 100})
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.ID.String() == strings.TrimSpace(id) {
			return s.Delete(ctx, id)
		}
	}
	return gorm.ErrRecordNotFound
}

func (s *InviteLinkService) ListForChat(ctx context.Context, chatID int64, limit int) (string, error) {
	records, err := s.List(ctx, InviteLinkListFilter{ChatID: chatID, Limit: limit})
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "本群暂无邀请链接。", nil
	}
	var builder strings.Builder
	builder.WriteString("本群邀请链接：")
	for _, record := range records {
		name := strings.TrimSpace(record.Name)
		if name == "" {
			name = "未命名"
		}
		builder.WriteString(fmt.Sprintf("\n%s %s\n加入：%d 次\n%s", record.ID.String(), name, record.JoinCount, record.InviteLink))
	}
	return builder.String(), nil
}

func (s *InviteLinkService) CreateForBot(ctx context.Context, chatID int64, name string, createsJoinRequest bool, operatorID int64) (bot.InviteLinkRecord, error) {
	record, err := s.Create(ctx, InviteLinkCreate{
		ChatID:             chatID,
		Name:               name,
		CreatesJoinRequest: createsJoinRequest,
		CreatedBy:          operatorID,
	})
	if err != nil {
		return bot.InviteLinkRecord{}, err
	}
	return bot.InviteLinkRecord{
		ID:                 record.ID.String(),
		ChatID:             record.ChatID,
		Name:               record.Name,
		InviteLink:         record.InviteLink,
		CreatesJoinRequest: record.CreatesJoinRequest,
		JoinCount:          record.JoinCount,
		CreatedBy:          record.CreatedBy,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}, nil
}

func (s *InviteLinkService) DeleteForChat(ctx context.Context, chatID int64, id string) error {
	records, err := s.List(ctx, InviteLinkListFilter{ChatID: chatID, Limit: 100})
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.ID.String() == strings.TrimSpace(id) {
			return s.Delete(ctx, id)
		}
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(id)); err == nil && parsed > 0 && parsed <= len(records) {
		return s.Delete(ctx, records[parsed-1].ID.String())
	}
	return gorm.ErrRecordNotFound
}
