package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	robfigcron "github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/config"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

func NewAPIDependencies(cfg config.Config, st *store.Store) api.Dependencies {
	admin := &adminAuthService{cfg: cfg, store: st}
	levels := NewLevelService(st)
	moderation := NewModerationService(st)
	autoReplies := NewAutoReplyService(st)
	return api.Dependencies{
		Auth:             admin,
		BotConfig:        &botConfigService{},
		Chats:            &chatBindingService{store: st},
		ChatPointConfigs: &chatPointConfigService{points: NewPointsService(st)},
		Points:           &pointsAPIService{points: NewPointsService(st)},
		Admin:            &adminAPIService{admin: NewAdminService(st, nil)},
		Lotteries:        &lotteryAPIService{lotteries: NewLotteryService(st), store: st},
		Levels:           &levelAPIService{levels: levels},
		AdminViolations:  &adminViolationAPIService{moderation: moderation},
		Keywords:         &keywordAPIService{moderation: moderation},
		AutoReplies:      &autoReplyAPIService{service: autoReplies},
		Backups:          &backupAPIService{backup: NewBackupService(st)},
		Templates:        &templateAPIService{templates: NewMessageTemplateService(st)},
		InviteLinks:      &inviteLinkAPIService{inviteLinks: NewInviteLinkService(st, cfg.Bot.Token)},
		Posts:            &postAPIService{store: st},
		Schedules:        &scheduleAPIService{store: st},
		Stats:            &statsAPIService{store: st},
		Users:            &userAPIService{store: st},
		Redis:            st.Redis,
		AllowedOriginSet: cfg.App.AllowedOrigins,
		EnableSwagger:    cfg.App.EnableSwagger,
		JWT: api.JWTConfig{
			SigningKey: cfg.JWT.Secret,
			Issuer:     cfg.JWT.Issuer,
			TTL:        cfg.JWT.AccessTokenTTL,
		},
	}
}

type adminAuthService struct {
	cfg   config.Config
	store *store.Store
}

func (s *adminAuthService) Authenticate(ctx context.Context, req api.AdminLoginRequest) (api.AdminIdentity, error) {
	username := req.Username
	if username == "" {
		username = req.Email
	}
	if subtleUsernameMismatch(username, s.cfg.App.AdminUsername) {
		return api.AdminIdentity{}, errors.New("invalid credentials")
	}
	if !verifyAdminPassword(req.Password, s.cfg.App.AdminPassword, s.cfg.App.AdminPasswordHash) {
		return api.AdminIdentity{}, errors.New("invalid credentials")
	}
	return api.AdminIdentity{
		ID:       "admin",
		Username: username,
		Email:    username,
		Role:     "super_admin",
		Name:     "Administrator",
		Language: "zh-CN",
	}, nil
}

func subtleUsernameMismatch(provided string, expected string) bool {
	provided = strings.TrimSpace(provided)
	expected = strings.TrimSpace(expected)
	return provided == "" || expected == "" || !strings.EqualFold(provided, expected)
}

func verifyAdminPassword(password string, plain string, hash string) bool {
	return api.VerifyConfiguredPassword(password, plain, hash)
}

func (s *adminAuthService) AuthenticateTelegram(ctx context.Context, req api.TelegramLoginRequest) (api.AdminIdentity, error) {
	loginData, err := VerifyTelegramLogin(map[string]string(req), s.cfg.Bot.Token)
	if err != nil {
		return api.AdminIdentity{}, err
	}
	displayName := strings.TrimSpace(strings.TrimSpace(loginData.FirstName) + " " + strings.TrimSpace(loginData.LastName))
	if displayName == "" {
		displayName = loginData.Username
	}
	user, err := upsertTelegramOwner(ctx, s.store, upsertTelegramOwnerInput{
		TelegramUserID: loginData.ID,
		Username:       loginData.Username,
		DisplayName:    displayName,
		Role:           "owner",
		PhotoURL:       loginData.PhotoURL,
	})
	if err != nil {
		return api.AdminIdentity{}, err
	}
	username := ""
	if user.Username != nil {
		username = *user.Username
	}
	return api.AdminIdentity{
		ID:             user.ID.String(),
		UserID:         user.ID.String(),
		TelegramUserID: loginData.ID,
		Username:       username,
		Role:           user.Role,
		Name:           user.DisplayName,
		DisplayName:    user.DisplayName,
		PhotoURL:       loginData.PhotoURL,
		Language:       user.LanguageCode,
	}, nil
}

type botConfigService struct{ config api.BotConfig }

func (s *botConfigService) Get(ctx context.Context) (*api.BotConfig, error) {
	if s.config.DefaultLanguage == "" {
		s.config = api.BotConfig{
			Enabled:             true,
			DefaultLanguage:     "zh-CN",
			TimeZone:            "Asia/Shanghai",
			AutoDeleteAfterSecs: 0,
			AllowForwardedPosts: true,
			EnableStatsTracking: true,
			EnablePoints:        true,
		}
	}
	return &s.config, nil
}

func (s *botConfigService) Update(ctx context.Context, req api.BotConfigUpdateRequest) (*api.BotConfig, error) {
	current, _ := s.Get(ctx)
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.DefaultLanguage != nil {
		current.DefaultLanguage = *req.DefaultLanguage
	}
	if req.TimeZone != nil {
		current.TimeZone = *req.TimeZone
	}
	if req.AutoDeleteEnabled != nil {
		current.AutoDeleteEnabled = *req.AutoDeleteEnabled
	}
	if req.AutoDeleteAfterSecs != nil {
		current.AutoDeleteAfterSecs = *req.AutoDeleteAfterSecs
	}
	if req.AllowForwardedPosts != nil {
		current.AllowForwardedPosts = *req.AllowForwardedPosts
	}
	if req.EnableStatsTracking != nil {
		current.EnableStatsTracking = *req.EnableStatsTracking
	}
	if req.EnablePoints != nil {
		current.EnablePoints = *req.EnablePoints
	}
	return current, nil
}

type chatBindingService struct{ store *store.Store }

func (s *chatBindingService) Bind(ctx context.Context, req api.ChatBindingRequest) (*api.ChatBinding, error) {
	if s.store == nil || s.store.DB == nil {
		return &api.ChatBinding{ChatID: req.ChatID, ChatType: req.ChatType, Title: req.Title, Username: req.Username, InviteLink: req.InviteLink, BoundBy: req.BoundBy, Description: req.Description, BoundAt: time.Now()}, nil
	}
	var owner *model.User
	if req.OwnerTelegramUserID != 0 {
		displayName := strings.TrimSpace(req.OwnerDisplayName)
		if displayName == "" {
			displayName = req.OwnerUsername
		}
		user, err := upsertTelegramOwner(ctx, s.store, upsertTelegramOwnerInput{
			TelegramUserID: req.OwnerTelegramUserID,
			Username:       strings.TrimPrefix(req.OwnerUsername, "@"),
			DisplayName:    displayName,
			Role:           "owner",
		})
		if err != nil {
			return nil, err
		}
		owner = user
	}
	title := req.Title
	username := req.Username
	inviteLink := req.InviteLink
	description := req.Description
	var chat model.TelegramChat
	err := s.store.DB.WithContext(ctx).Where("telegram_chat_id = ?", req.ChatID).First(&chat).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil && owner != nil && chat.OwnerUserID != nil && *chat.OwnerUserID != owner.ID {
		return nil, fmt.Errorf("该群已被其他群主绑定")
	}
	updates := model.TelegramChat{
		TelegramChatID: req.ChatID,
		Type:           req.ChatType,
		Title:          &title,
		Username:       stringPtrOrNil(username),
		InviteLink:     stringPtrOrNil(inviteLink),
		Description:    stringPtrOrNil(description),
		Status:         "active",
		LastSeenAt:     timePtr(time.Now()),
	}
	if owner != nil {
		updates.OwnerUserID = &owner.ID
	}
	if err := s.store.DB.WithContext(ctx).Where("telegram_chat_id = ?", req.ChatID).Assign(updates).FirstOrCreate(&chat).Error; err != nil {
		return nil, err
	}
	if owner != nil {
		if err := s.ensureChatOwnerAdmin(ctx, chat, *owner); err != nil {
			return nil, err
		}
	}
	return chatBindingToAPI(chat, req.BoundBy), nil
}

func (s *chatBindingService) Unbind(ctx context.Context, chatID int64) error {
	if s.store == nil || s.store.DB == nil {
		return nil
	}
	return s.store.DB.WithContext(ctx).Model(&model.TelegramChat{}).Where("telegram_chat_id = ?", chatID).Update("status", "inactive").Error
}

func (s *chatBindingService) List(ctx context.Context, query api.CommonListQuery) ([]api.ChatBinding, error) {
	if s.store == nil || s.store.DB == nil {
		return []api.ChatBinding{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.TelegramChat{})
	if strings.TrimSpace(query.OwnerUserID) != "" {
		if ownerID, err := uuid.Parse(strings.TrimSpace(query.OwnerUserID)); err == nil {
			db = db.Where("owner_user_id = ?", ownerID)
		} else {
			return []api.ChatBinding{}, nil
		}
	}
	var chats []model.TelegramChat
	if err := db.Order("created_at desc").Limit(normalLimit(query.Limit)).Offset(query.Offset).Find(&chats).Error; err != nil {
		return nil, err
	}
	out := make([]api.ChatBinding, 0, len(chats))
	for _, chat := range chats {
		out = append(out, *chatBindingToAPI(chat, ""))
	}
	return out, nil
}

func (s *chatBindingService) ListByTelegramUser(ctx context.Context, telegramUserID int64, limit int) ([]api.ChatBinding, error) {
	if s.store == nil || s.store.DB == nil || telegramUserID == 0 {
		return []api.ChatBinding{}, nil
	}
	var user model.User
	if err := s.store.DB.WithContext(ctx).Where("telegram_user_id = ?", telegramUserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []api.ChatBinding{}, nil
		}
		return nil, err
	}
	return s.List(ctx, api.CommonListQuery{OwnerUserID: user.ID.String(), Limit: limit})
}
func (s *chatBindingService) UserOwnsChat(ctx context.Context, userID string, chatID string) (bool, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return true, nil
	}
	ownerID, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return false, nil
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return true, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.TelegramChat{}).Where("owner_user_id = ?", ownerID)
	if tgChatID, err := strconv.ParseInt(chatID, 10, 64); err == nil {
		db = db.Where("telegram_chat_id = ?", tgChatID)
	} else if parsed, err := uuid.Parse(chatID); err == nil {
		db = db.Where("id = ?", parsed)
	} else {
		return false, nil
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *chatBindingService) ensureChatOwnerAdmin(ctx context.Context, chat model.TelegramChat, owner model.User) error {
	now := time.Now()
	admin := model.ChatAdmin{
		ChatID:          chat.ID,
		UserID:          owner.ID,
		Role:            "admin",
		CanManage:       true,
		CanPost:         true,
		CanDelete:       true,
		CanBan:          true,
		GrantedByUserID: &owner.ID,
		GrantedAt:       now,
	}
	return s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chat.ID, owner.ID).
		Assign(admin).
		FirstOrCreate(&admin).Error
}

type chatPointConfigService struct{ points *PointsService }

func (s *chatPointConfigService) Get(ctx context.Context, chatID int64) (*api.ChatPointConfig, error) {
	if s == nil || s.points == nil {
		cfg := bot.ChatPointConfig{ChatID: chatID, PointText: 1, PointPhoto: 3, PointSticker: 2, PointVideo: 3, PointFile: 2, PointVoice: 3, CooldownSeconds: 60, Enabled: true}
		return botChatPointConfigToAPI(cfg), nil
	}
	config, err := s.points.GetConfig(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return botChatPointConfigToAPI(config), nil
}

func (s *chatPointConfigService) Update(ctx context.Context, chatID int64, req api.ChatPointConfigUpdateRequest) (*api.ChatPointConfig, error) {
	if s == nil || s.points == nil {
		cfg := bot.ChatPointConfig{ChatID: chatID, PointText: 1, PointPhoto: 3, PointSticker: 2, PointVideo: 3, PointFile: 2, PointVoice: 3, CooldownSeconds: 60, Enabled: true}
		if req.PointText != nil {
			cfg.PointText = *req.PointText
		}
		if req.PointPhoto != nil {
			cfg.PointPhoto = *req.PointPhoto
		}
		if req.PointSticker != nil {
			cfg.PointSticker = *req.PointSticker
		}
		if req.PointVideo != nil {
			cfg.PointVideo = *req.PointVideo
		}
		if req.PointFile != nil {
			cfg.PointFile = *req.PointFile
		}
		if req.PointVoice != nil {
			cfg.PointVoice = *req.PointVoice
		}
		if req.CooldownSeconds != nil {
			cfg.CooldownSeconds = *req.CooldownSeconds
		}
		if req.Enabled != nil {
			cfg.Enabled = *req.Enabled
		}
		return botChatPointConfigToAPI(cfg), nil
	}

	config, err := s.points.UpdateConfig(ctx, chatID, bot.ChatPointConfigPatch{
		PointText:       req.PointText,
		PointPhoto:      req.PointPhoto,
		PointSticker:    req.PointSticker,
		PointVideo:      req.PointVideo,
		PointFile:       req.PointFile,
		PointVoice:      req.PointVoice,
		CooldownSeconds: req.CooldownSeconds,
		Enabled:         req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	return botChatPointConfigToAPI(config), nil
}

type pointsAPIService struct{ points *PointsService }

func (s *pointsAPIService) GetRank(ctx context.Context, chatID int64, period string, limit int) ([]api.PointRankItem, error) {
	if s == nil || s.points == nil {
		return []api.PointRankItem{}, nil
	}
	entries, err := s.points.GetRankEntries(ctx, chatID, period, limit)
	if err != nil {
		return nil, err
	}
	out := make([]api.PointRankItem, 0, len(entries))
	for _, entry := range entries {
		out = append(out, api.PointRankItem{
			Rank:   entry.Rank,
			UserID: entry.UserID,
			Points: entry.Points,
		})
	}
	return out, nil
}

func (s *pointsAPIService) GetUser(ctx context.Context, chatID int64, userID int64) (*api.PointUserResponse, error) {
	if s == nil || s.points == nil {
		return &api.PointUserResponse{ChatID: chatID, UserID: userID}, nil
	}
	point, err := s.points.GetUserPoint(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	return userPointToAPI(point), nil
}

func (s *pointsAPIService) AdjustUser(ctx context.Context, chatID int64, userID int64, delta int, reason string) (*api.PointUserResponse, error) {
	if s == nil || s.points == nil {
		return &api.PointUserResponse{ChatID: chatID, UserID: userID, TotalPoints: int64(delta)}, nil
	}
	point, err := s.points.AdjustUserPoints(ctx, chatID, userID, delta, reason)
	if err != nil {
		return nil, err
	}
	return userPointToAPI(point), nil
}

func (s *pointsAPIService) ListLogs(ctx context.Context, chatID int64, userID int64, query api.PointLogListQuery) (*api.PointLogListResponse, error) {
	if s == nil || s.points == nil {
		return &api.PointLogListResponse{Items: []api.PointLogItem{}}, nil
	}
	logs, nextCursor, err := s.points.ListPointLogs(ctx, chatID, userID, query.Limit, query.Offset, query.Cursor)
	if err != nil {
		return nil, err
	}
	out := make([]api.PointLogItem, 0, len(logs))
	for _, log := range logs {
		out = append(out, pointLogToAPI(log))
	}
	return &api.PointLogListResponse{Items: out, NextCursor: nextCursor}, nil
}

type lotteryAPIService struct {
	lotteries *LotteryService
	store     *store.Store
}

func (s *lotteryAPIService) List(ctx context.Context, query api.LotteryListQuery) ([]api.Lottery, error) {
	if s == nil || s.lotteries == nil {
		return []api.Lottery{}, nil
	}
	return s.lotteries.List(ctx, query)
}

func (s *lotteryAPIService) Create(ctx context.Context, req api.LotteryCreateRequest) (*api.Lottery, error) {
	if s == nil || s.lotteries == nil {
		return nil, gorm.ErrInvalidDB
	}
	return s.lotteries.Create(ctx, req)
}

func (s *lotteryAPIService) Cancel(ctx context.Context, id int64, ownerUserID string) error {
	chatID, err := lotteryChatID(ctx, s.store, id)
	if err != nil {
		return err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, chatID); err != nil {
		return err
	}
	return s.lotteries.Cancel(ctx, id)
}

func (s *lotteryAPIService) Entries(ctx context.Context, id int64, ownerUserID string) ([]api.LotteryEntry, error) {
	chatID, err := lotteryChatID(ctx, s.store, id)
	if err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, chatID); err != nil {
		return nil, err
	}
	return s.lotteries.Entries(ctx, id)
}

func (s *lotteryAPIService) Winners(ctx context.Context, id int64, ownerUserID string) ([]api.LotteryEntry, error) {
	chatID, err := lotteryChatID(ctx, s.store, id)
	if err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, chatID); err != nil {
		return nil, err
	}
	return s.lotteries.Winners(ctx, id)
}

type levelAPIService struct {
	levels *LevelService
}

func (s *levelAPIService) List(ctx context.Context, query api.LevelListQuery) ([]api.Level, error) {
	if s == nil || s.levels == nil || s.levels.store == nil || s.levels.store.DB == nil {
		return []api.Level{}, nil
	}
	db := s.levels.store.DB.WithContext(ctx).Model(&model.LevelConfig{})
	ownedIDs, err := ownedTelegramChatIDs(ctx, s.levels.store, query.OwnerUserID)
	if err != nil {
		return nil, err
	}
	db = scopeTelegramChatID(db, "chat_id", query.ChatID, ownedIDs)
	var records []model.LevelConfig
	err = db.Order("chat_id asc, min_points asc, level asc").
		Limit(normalLimit(query.Limit)).
		Offset(query.Offset).
		Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []api.Level{}, nil
		}
		return nil, err
	}
	out := make([]api.Level, 0, len(records))
	for _, record := range records {
		out = append(out, levelConfigToAPI(record))
	}
	return out, nil
}

func (s *levelAPIService) Create(ctx context.Context, req api.LevelCreateRequest) (*api.Level, error) {
	if s == nil || s.levels == nil || s.levels.store == nil || s.levels.store.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	level := req.Level
	if level <= 0 {
		var maxLevel int
		err := s.levels.store.DB.WithContext(ctx).
			Unscoped().
			Model(&model.LevelConfig{}).
			Where("chat_id = ?", req.ChatID).
			Select("COALESCE(MAX(level), 0)").
			Scan(&maxLevel).Error
		if err != nil && !isMissingTableError(err) {
			return nil, err
		}
		level = maxLevel + 1
	}
	now := time.Now()
	record := model.LevelConfig{
		BaseModel:    model.BaseModel{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
		ChatID:       req.ChatID,
		Level:        level,
		MinPoints:    req.MinPoints,
		Label:        strings.TrimSpace(req.Name),
		Badge:        strings.TrimSpace(req.Badge),
		CanPostLink:  hasPermission(req.Permissions, "post_link", true),
		CanPostMedia: hasPermission(req.Permissions, "post_media", true),
	}
	err := s.levels.store.DB.WithContext(ctx).Create(&record).Error
	if err != nil {
		if isMissingTableError(err) {
			return nil, err
		}
		return nil, err
	}
	item := levelConfigToAPI(record)
	return &item, nil
}

func (s *levelAPIService) Update(ctx context.Context, id string, req api.LevelUpdateRequest, ownerUserID string) (*api.Level, error) {
	if s == nil || s.levels == nil || s.levels.store == nil || s.levels.store.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	var record model.LevelConfig
	if err := s.findLevelConfig(ctx, id, req.ChatID, &record); err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.levels.store, ownerUserID, record.ChatID); err != nil {
		return nil, err
	}
	updates := map[string]any{"updated_at": time.Now()}
	if req.Level != nil {
		updates["level"] = *req.Level
	}
	if req.Name != nil {
		updates["label"] = strings.TrimSpace(*req.Name)
	}
	if req.MinPoints != nil {
		updates["min_points"] = *req.MinPoints
	}
	if req.Badge != nil {
		updates["badge"] = strings.TrimSpace(*req.Badge)
	}
	if req.Permissions != nil {
		updates["can_post_link"] = hasPermission(*req.Permissions, "post_link", false)
		updates["can_post_media"] = hasPermission(*req.Permissions, "post_media", false)
	}
	if err := s.levels.store.DB.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
		if isMissingTableError(err) {
			return nil, err
		}
		return nil, err
	}
	if err := s.levels.store.DB.WithContext(ctx).First(&record, "id = ?", record.ID).Error; err != nil {
		return nil, err
	}
	item := levelConfigToAPI(record)
	return &item, nil
}

func (s *levelAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	if s == nil || s.levels == nil || s.levels.store == nil || s.levels.store.DB == nil {
		return nil
	}
	db := s.levels.store.DB.WithContext(ctx)
	if parsed, err := uuid.Parse(strings.TrimSpace(id)); err == nil {
		var record model.LevelConfig
		if err := db.First(&record, "id = ?", parsed).Error; err != nil {
			if isMissingTableError(err) || errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if err := ensureOwnedTelegramChatID(ctx, s.levels.store, ownerUserID, record.ChatID); err != nil {
			return err
		}
		err = db.Delete(&record).Error
		if isMissingTableError(err) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(ownerUserID) != "" {
		return api.ErrForbidden
	}
	level, err := strconv.Atoi(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	err = db.Where("level = ?", level).Delete(&model.LevelConfig{}).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func (s *levelAPIService) findLevelConfig(ctx context.Context, id string, chatID int64, dest *model.LevelConfig) error {
	id = strings.TrimSpace(id)
	db := s.levels.store.DB.WithContext(ctx)
	if parsed, err := uuid.Parse(id); err == nil {
		return db.First(dest, "id = ?", parsed).Error
	}
	level, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	query := db.Where("level = ?", level)
	if chatID != 0 {
		query = query.Where("chat_id = ?", chatID)
	}
	return query.Order("chat_id asc").First(dest).Error
}

type adminViolationAPIService struct {
	moderation *ModerationService
}

func (s *adminViolationAPIService) List(ctx context.Context, query api.AdminViolationListQuery) (*api.CursorListResponse[api.AdminViolation], error) {
	if s == nil || s.moderation == nil {
		return &api.CursorListResponse[api.AdminViolation]{Items: []api.AdminViolation{}}, nil
	}
	chatIDs, scoped, err := queryTelegramChatIDs(ctx, s.moderation.store, query.OwnerUserID, query.ChatID)
	if err != nil {
		return nil, err
	}
	if scoped && len(chatIDs) == 0 {
		return &api.CursorListResponse[api.AdminViolation]{Items: []api.AdminViolation{}}, nil
	}
	records, err := s.moderation.ListViolationsFiltered(ctx, ViolationListFilter{
		ChatID:  query.ChatID,
		ChatIDs: chatIDs,
		UserID:  query.UserID,
		Type:    query.Type,
		Status:  query.Status,
		Limit:   query.Limit + 1,
		Offset:  query.Offset,
		Cursor:  query.Cursor,
	})
	if err != nil {
		return nil, err
	}
	items := records
	nextCursor := ""
	if len(records) > query.Limit && query.Limit > 0 {
		items = records[:query.Limit]
		last := items[len(items)-1]
		nextCursor = encodeUUIDCursor(last.CreatedAt, last.ID)
	}
	out := make([]api.AdminViolation, 0, len(items))
	for _, record := range items {
		out = append(out, violationRecordToAPI(record))
	}
	return &api.CursorListResponse[api.AdminViolation]{Items: out, NextCursor: nextCursor}, nil
}
func (s *adminViolationAPIService) Update(ctx context.Context, id string, req api.AdminViolationUpdateRequest, ownerUserID string) (*api.AdminViolation, error) {
	if s == nil || s.moderation == nil {
		return nil, gorm.ErrInvalidDB
	}
	if s.moderation.store == nil || s.moderation.store.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	var current model.ViolationRecord
	if err := s.moderation.store.DB.WithContext(ctx).Select("chat_id").First(&current, "id = ?", parsed).Error; err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.moderation.store, ownerUserID, current.ChatID); err != nil {
		return nil, err
	}
	record, err := s.moderation.UpdateViolation(ctx, id, req.Status, req.Resolution)
	if err != nil {
		return nil, err
	}
	item := violationRecordToAPI(record)
	return &item, nil
}

type keywordAPIService struct {
	moderation *ModerationService
}

func (s *keywordAPIService) List(ctx context.Context, query api.KeywordListQuery) ([]api.Keyword, error) {
	if s == nil || s.moderation == nil {
		return []api.Keyword{}, nil
	}
	if chatIDs, scoped, err := queryTelegramChatIDs(ctx, s.moderation.store, query.OwnerUserID, query.ChatID); scoped {
		if err != nil {
			return nil, err
		}
		if len(chatIDs) == 0 {
			return []api.Keyword{}, nil
		}
		var out []api.Keyword
		for _, chatID := range chatIDs {
			records, err := s.moderation.ListKeywordFilters(ctx, KeywordFilterListFilter{
				ChatID:  chatID,
				Scope:   query.Scope,
				Action:  query.Action,
				Enabled: query.Enabled,
				Limit:   query.Limit,
				Offset:  query.Offset,
			})
			if err != nil {
				return nil, err
			}
			for _, record := range records {
				out = append(out, keywordFilterToAPI(record))
			}
		}
		return out, nil
	}
	records, err := s.moderation.ListKeywordFilters(ctx, KeywordFilterListFilter{
		ChatID:  query.ChatID,
		Scope:   query.Scope,
		Action:  query.Action,
		Enabled: query.Enabled,
		Limit:   query.Limit,
		Offset:  query.Offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]api.Keyword, 0, len(records))
	for _, record := range records {
		out = append(out, keywordFilterToAPI(record))
	}
	return out, nil
}

func (s *keywordAPIService) Create(ctx context.Context, req api.KeywordCreateRequest) (*api.Keyword, error) {
	if s == nil || s.moderation == nil {
		return nil, gorm.ErrInvalidDB
	}
	record, err := s.moderation.CreateKeywordFilter(ctx, KeywordFilterCreate{
		ChatID:    req.ChatID,
		Keyword:   requestKeyword(req.Pattern, req.Keyword),
		MatchType: req.MatchType,
		Action:    req.Action,
		Scope:     req.Scope,
		ReplyText: req.ReplyText,
		Enabled:   req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	item := keywordFilterToAPI(record)
	return &item, nil
}

func (s *keywordAPIService) Update(ctx context.Context, id string, req api.KeywordUpdateRequest, ownerUserID string) (*api.Keyword, error) {
	if s == nil || s.moderation == nil {
		return nil, gorm.ErrInvalidDB
	}
	chatID, err := keywordFilterChatID(ctx, s.moderation.store, id)
	if err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.moderation.store, ownerUserID, chatID); err != nil {
		return nil, err
	}
	record, err := s.moderation.UpdateKeywordFilter(ctx, id, KeywordFilterPatch{
		Keyword:   firstStringPtr(req.Keyword, req.Pattern),
		MatchType: req.MatchType,
		Action:    req.Action,
		Scope:     req.Scope,
		ReplyText: req.ReplyText,
		Enabled:   req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	item := keywordFilterToAPI(record)
	return &item, nil
}

func (s *keywordAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	if s == nil || s.moderation == nil {
		return nil
	}
	chatID, err := keywordFilterChatID(ctx, s.moderation.store, id)
	if err != nil {
		return err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.moderation.store, ownerUserID, chatID); err != nil {
		return err
	}
	return s.moderation.DeleteKeywordFilter(ctx, id)
}

type autoReplyAPIService struct {
	service *AutoReplyService
}

func (s *autoReplyAPIService) List(ctx context.Context, query api.AutoReplyListQuery) ([]api.AutoReply, error) {
	if s == nil || s.service == nil {
		return []api.AutoReply{}, nil
	}
	if chatIDs, scoped, err := queryTelegramChatIDs(ctx, s.service.store, query.OwnerUserID, query.ChatID); scoped {
		if err != nil {
			return nil, err
		}
		if len(chatIDs) == 0 {
			return []api.AutoReply{}, nil
		}
		var out []api.AutoReply
		for _, chatID := range chatIDs {
			records, err := s.service.List(ctx, AutoReplyListFilter{ChatID: chatID, Enabled: query.Enabled, Limit: query.Limit, Offset: query.Offset})
			if err != nil {
				return nil, err
			}
			for _, record := range records {
				out = append(out, autoReplyRecordToAPI(record))
			}
		}
		return out, nil
	}
	records, err := s.service.List(ctx, AutoReplyListFilter{
		ChatID:  query.ChatID,
		Enabled: query.Enabled,
		Limit:   query.Limit,
		Offset:  query.Offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]api.AutoReply, 0, len(records))
	for _, record := range records {
		out = append(out, autoReplyRecordToAPI(record))
	}
	return out, nil
}

func (s *autoReplyAPIService) Create(ctx context.Context, req api.AutoReplyCreateRequest) (*api.AutoReply, error) {
	if s == nil || s.service == nil {
		return nil, gorm.ErrInvalidDB
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	record, err := s.service.Create(ctx, &model.AutoReply{
		ChatID:    req.ChatID,
		Keyword:   req.Keyword,
		MatchType: req.MatchType,
		ReplyText: req.ReplyText,
		Enabled:   enabled,
		CreatedBy: req.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	item := autoReplyRecordToAPI(record)
	return &item, nil
}

func (s *autoReplyAPIService) Update(ctx context.Context, id string, req api.AutoReplyUpdateRequest, ownerUserID string) (*api.AutoReply, error) {
	if s == nil || s.service == nil {
		return nil, gorm.ErrInvalidDB
	}
	chatID, err := autoReplyChatID(ctx, s.service.store, id)
	if err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.service.store, ownerUserID, chatID); err != nil {
		return nil, err
	}
	record, err := s.service.Update(ctx, id, AutoReplyPatch{
		Keyword:   req.Keyword,
		MatchType: req.MatchType,
		ReplyText: req.ReplyText,
		Enabled:   req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	item := autoReplyRecordToAPI(record)
	return &item, nil
}

func (s *autoReplyAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	if s == nil || s.service == nil {
		return nil
	}
	chatID, err := autoReplyChatID(ctx, s.service.store, id)
	if err != nil {
		return err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.service.store, ownerUserID, chatID); err != nil {
		return err
	}
	return s.service.Delete(ctx, id)
}

type backupAPIService struct {
	backup *BackupService
}

func (s *backupAPIService) Export(ctx context.Context, scope string) (*api.BackupData, error) {
	if s == nil || s.backup == nil {
		return nil, gorm.ErrInvalidDB
	}
	data, err := s.backup.Export(ctx, scope)
	if err != nil {
		return nil, err
	}
	return &api.BackupData{
		Version:    data.Version,
		ExportedAt: data.ExportedAt,
		Scope:      data.Scope,
		Tables:     data.Tables,
	}, nil
}

func (s *backupAPIService) Import(ctx context.Context, data *api.BackupData, mode string) error {
	if s == nil || s.backup == nil {
		return gorm.ErrInvalidDB
	}
	if data == nil {
		return errors.New("backup data is required")
	}
	return s.backup.Import(ctx, &BackupData{
		Version:    data.Version,
		ExportedAt: data.ExportedAt,
		Scope:      data.Scope,
		Tables:     data.Tables,
	}, mode)
}

type templateAPIService struct {
	templates *MessageTemplateService
}

func (s *templateAPIService) List(ctx context.Context, query api.TemplateListQuery) (*api.CursorListResponse[api.Template], error) {
	if s == nil || s.templates == nil {
		return &api.CursorListResponse[api.Template]{Items: []api.Template{}}, nil
	}
	if chatIDs, scoped, err := queryTelegramChatIDs(ctx, s.templates.store, query.OwnerUserID, query.ChatID); scoped && query.ChatID == 0 {
		if err != nil {
			return nil, err
		}
		if len(chatIDs) == 0 {
			return &api.CursorListResponse[api.Template]{Items: []api.Template{}}, nil
		}
		if len(chatIDs) > 1 && strings.TrimSpace(query.Cursor) != "" {
			return nil, fmt.Errorf("cursor pagination requires chatID filter")
		}
		var out []api.Template
		for _, chatID := range chatIDs {
			records, err := s.templates.List(ctx, MessageTemplateListFilter{ChatID: chatID, Limit: query.Limit + 1, Offset: query.Offset})
			if err != nil {
				return nil, err
			}
			for _, record := range records {
				out = append(out, messageTemplateToAPI(record))
			}
		}
		nextCursor := ""
		if len(out) > query.Limit && query.Limit > 0 {
			trimmed := out[:query.Limit]
			last := trimmed[len(trimmed)-1]
			parsedID, err := uuid.Parse(last.ID)
			if err != nil {
				return nil, err
			}
			nextCursor = encodeUUIDCursor(last.CreatedAt, parsedID)
			out = trimmed
		}
		return &api.CursorListResponse[api.Template]{Items: out, NextCursor: nextCursor}, nil
	}
	records, err := s.templates.List(ctx, MessageTemplateListFilter{ChatID: query.ChatID, Limit: query.Limit + 1, Offset: query.Offset, Cursor: query.Cursor})
	if err != nil {
		return nil, err
	}
	items := records
	nextCursor := ""
	if len(records) > query.Limit && query.Limit > 0 {
		items = records[:query.Limit]
		last := items[len(items)-1]
		nextCursor = encodeUUIDCursor(last.CreatedAt, last.ID)
	}
	out := make([]api.Template, 0, len(items))
	for _, record := range items {
		out = append(out, messageTemplateToAPI(record))
	}
	return &api.CursorListResponse[api.Template]{Items: out, NextCursor: nextCursor}, nil
}

func (s *templateAPIService) Create(ctx context.Context, req api.TemplateCreateRequest, ownerUserID string) (*api.Template, error) {
	if s == nil || s.templates == nil {
		return nil, gorm.ErrInvalidDB
	}
	if err := ensureOwnedOptionalTelegramChatID(ctx, s.templates.store, ownerUserID, req.ChatID); err != nil {
		return nil, err
	}
	record, err := s.templates.Create(ctx, MessageTemplateCreate{
		ChatID:    req.ChatID,
		Name:      req.Name,
		Content:   req.Content,
		MediaType: req.MediaType,
		MediaURL:  req.MediaURL,
		ParseMode: req.ParseMode,
		CreatedBy: req.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	item := messageTemplateToAPI(record)
	return &item, nil
}

func (s *templateAPIService) Update(ctx context.Context, id string, req api.TemplateUpdateRequest, ownerUserID string) (*api.Template, error) {
	if s == nil || s.templates == nil {
		return nil, gorm.ErrInvalidDB
	}
	chatID, err := messageTemplateChatID(ctx, s.templates.store, id)
	if err != nil {
		return nil, err
	}
	if err := ensureOwnedOptionalTelegramChatID(ctx, s.templates.store, ownerUserID, chatID); err != nil {
		return nil, err
	}
	if req.ChatID != nil {
		if err := ensureOwnedOptionalTelegramChatID(ctx, s.templates.store, ownerUserID, *req.ChatID); err != nil {
			return nil, err
		}
	}
	record, err := s.templates.Update(ctx, id, MessageTemplatePatch{
		ChatID:    req.ChatID,
		Name:      req.Name,
		Content:   req.Content,
		MediaType: req.MediaType,
		MediaURL:  req.MediaURL,
		ParseMode: req.ParseMode,
	})
	if err != nil {
		return nil, err
	}
	item := messageTemplateToAPI(record)
	return &item, nil
}

func (s *templateAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	if s == nil || s.templates == nil {
		return nil
	}
	chatID, err := messageTemplateChatID(ctx, s.templates.store, id)
	if err != nil {
		return err
	}
	if err := ensureOwnedOptionalTelegramChatID(ctx, s.templates.store, ownerUserID, chatID); err != nil {
		return err
	}
	return s.templates.Delete(ctx, id)
}

type inviteLinkAPIService struct {
	inviteLinks *InviteLinkService
}

func (s *inviteLinkAPIService) List(ctx context.Context, query api.InviteLinkListQuery) (*api.CursorListResponse[api.InviteLink], error) {
	if s == nil || s.inviteLinks == nil {
		return &api.CursorListResponse[api.InviteLink]{Items: []api.InviteLink{}}, nil
	}
	if chatIDs, scoped, err := queryTelegramChatIDs(ctx, s.inviteLinks.store, query.OwnerUserID, query.ChatID); scoped && query.ChatID == 0 {
		if err != nil {
			return nil, err
		}
		if len(chatIDs) == 0 {
			return &api.CursorListResponse[api.InviteLink]{Items: []api.InviteLink{}}, nil
		}
		if len(chatIDs) > 1 && strings.TrimSpace(query.Cursor) != "" {
			return nil, fmt.Errorf("cursor pagination requires chatID filter")
		}
		var out []api.InviteLink
		for _, chatID := range chatIDs {
			records, err := s.inviteLinks.List(ctx, InviteLinkListFilter{ChatID: chatID, Limit: query.Limit + 1, Offset: query.Offset})
			if err != nil {
				return nil, err
			}
			for _, record := range records {
				out = append(out, inviteLinkToAPI(record))
			}
		}
		nextCursor := ""
		if len(out) > query.Limit && query.Limit > 0 {
			trimmed := out[:query.Limit]
			last := trimmed[len(trimmed)-1]
			parsedID, err := uuid.Parse(last.ID)
			if err != nil {
				return nil, err
			}
			nextCursor = encodeUUIDCursor(last.CreatedAt, parsedID)
			out = trimmed
		}
		return &api.CursorListResponse[api.InviteLink]{Items: out, NextCursor: nextCursor}, nil
	}
	records, err := s.inviteLinks.List(ctx, InviteLinkListFilter{ChatID: query.ChatID, Limit: query.Limit + 1, Offset: query.Offset, Cursor: query.Cursor})
	if err != nil {
		return nil, err
	}
	items := records
	nextCursor := ""
	if len(records) > query.Limit && query.Limit > 0 {
		items = records[:query.Limit]
		last := items[len(items)-1]
		nextCursor = encodeUUIDCursor(last.CreatedAt, last.ID)
	}
	out := make([]api.InviteLink, 0, len(items))
	for _, record := range items {
		out = append(out, inviteLinkToAPI(record))
	}
	return &api.CursorListResponse[api.InviteLink]{Items: out, NextCursor: nextCursor}, nil
}

func (s *inviteLinkAPIService) Create(ctx context.Context, req api.InviteLinkCreateRequest) (*api.InviteLink, error) {
	if s == nil || s.inviteLinks == nil {
		return nil, gorm.ErrInvalidDB
	}
	record, err := s.inviteLinks.Create(ctx, InviteLinkCreate{
		ChatID:             req.ChatID,
		Name:               req.Name,
		CreatesJoinRequest: req.CreatesJoinRequest,
		CreatedBy:          req.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	item := inviteLinkToAPI(record)
	return &item, nil
}

func (s *inviteLinkAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	if s == nil || s.inviteLinks == nil {
		return nil
	}
	chatID, err := inviteLinkChatID(ctx, s.inviteLinks.store, id)
	if err != nil {
		return err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.inviteLinks.store, ownerUserID, chatID); err != nil {
		return err
	}
	return s.inviteLinks.Delete(ctx, id)
}

type postAPIService struct{ store *store.Store }

func (s *postAPIService) Create(ctx context.Context, req api.PostCreateRequest) (*api.Post, error) {
	now := time.Now()
	runOnceAt := req.RunOnceAt
	if runOnceAt == nil {
		runOnceAt = req.PublishAt
	}
	req.CronExpr = strings.TrimSpace(req.CronExpr)
	req.MediaType = normalizeScheduledPostMediaType(req.MediaType)
	if err := validateScheduledPostCreate(req, runOnceAt); err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if s.store == nil || s.store.DB == nil {
		return &api.Post{
			ID:        strconv.FormatUint(uint64(now.UnixNano()), 10),
			ChatID:    req.ChatID,
			Title:     req.Title,
			Content:   req.Content,
			MediaURL:  req.MediaURL,
			MediaType: req.MediaType,
			CronExpr:  req.CronExpr,
			RunOnceAt: runOnceAt,
			Enabled:   enabled,
			PublishAt: runOnceAt,
			Status:    scheduledPostStatus(enabled),
			Language:  req.Language,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}
	post := model.ScheduledPost{
		ChatID:            req.ChatID,
		Title:             req.Title,
		Content:           req.Content,
		MediaURL:          req.MediaURL,
		MediaType:         req.MediaType,
		CronExpr:          req.CronExpr,
		RunOnceAt:         runOnceAt,
		Enabled:           enabled,
		CreatedAt:         now,
		AutoDeleteSeconds: req.AutoDeleteSeconds,
	}
	if req.PinAfterSend != nil {
		post.PinAfterSend = *req.PinAfterSend
	}
	if err := s.store.DB.WithContext(ctx).Select("*").Create(&post).Error; err != nil {
		return nil, err
	}
	if !enabled {
		if err := s.store.DB.WithContext(ctx).Model(&model.ScheduledPost{}).Where("id = ?", post.ID).Update("enabled", false).Error; err != nil {
			return nil, err
		}
		post.Enabled = false
	}
	item := modelScheduledPostToAPI(post)
	return &item, nil
}

func (s *postAPIService) List(ctx context.Context, query api.CommonListQuery) ([]api.Post, error) {
	if s.store == nil || s.store.DB == nil {
		return []api.Post{}, nil
	}
	var posts []model.ScheduledPost
	db := s.store.DB.WithContext(ctx).Model(&model.ScheduledPost{})
	ownedIDs, err := ownedTelegramChatIDs(ctx, s.store, query.OwnerUserID)
	if err != nil {
		return nil, err
	}
	db = scopeTelegramChatID(db, "chat_id", 0, ownedIDs)
	limit := normalLimit(query.Limit)
	if strings.TrimSpace(query.Cursor) != "" {
		cursorTime, cursorID, err := decodePostCursor(query.Cursor)
		if err != nil {
			return nil, err
		}
		db = db.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorTime, cursorTime, cursorID)
	}
	queryExec := db.Order("created_at desc, id desc").Limit(limit)
	if strings.TrimSpace(query.Cursor) == "" {
		queryExec = queryExec.Offset(query.Offset)
	}
	if err := queryExec.Find(&posts).Error; err != nil {
		return nil, err
	}
	out := make([]api.Post, 0, len(posts))
	for _, post := range posts {
		out = append(out, modelScheduledPostToAPI(post))
	}
	return out, nil
}

func (s *postAPIService) Get(ctx context.Context, id string, ownerUserID string) (*api.Post, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	parsed, err := parseScheduledPostID(id)
	if err != nil {
		return nil, err
	}
	var post model.ScheduledPost
	if err := s.store.DB.WithContext(ctx).First(&post, "id = ?", parsed).Error; err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, post.ChatID); err != nil {
		return nil, err
	}
	item := modelScheduledPostToAPI(post)
	return &item, nil
}

func (s *postAPIService) Update(ctx context.Context, id string, req api.PostUpdateRequest, ownerUserID string) (*api.Post, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	parsed, err := parseScheduledPostID(id)
	if err != nil {
		return nil, err
	}
	var post model.ScheduledPost
	if err := s.store.DB.WithContext(ctx).First(&post, "id = ?", parsed).Error; err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, post.ChatID); err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.MediaURL != nil {
		updates["media_url"] = *req.MediaURL
	}
	if req.MediaType != nil {
		updates["media_type"] = normalizeScheduledPostMediaType(*req.MediaType)
	}
	if req.CronExpr != nil {
		updates["cron_expr"] = strings.TrimSpace(*req.CronExpr)
	}
	runOnceAt := req.RunOnceAt
	if runOnceAt == nil {
		runOnceAt = req.PublishAt
	}
	if runOnceAt != nil {
		updates["run_once_at"] = *runOnceAt
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Status != nil {
		updates["enabled"] = strings.EqualFold(*req.Status, "enabled") || strings.EqualFold(*req.Status, "active")
	}
	if req.PinAfterSend != nil {
		updates["pin_after_send"] = *req.PinAfterSend
	}
	if req.AutoDeleteSeconds != nil {
		updates["auto_delete_seconds"] = *req.AutoDeleteSeconds
	}
	if len(updates) > 0 {
		if err := s.store.DB.WithContext(ctx).Model(&model.ScheduledPost{}).Where("id = ?", parsed).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, id, ownerUserID)
}

func (s *postAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	if s.store == nil || s.store.DB == nil {
		return nil
	}
	parsed, err := parseScheduledPostID(id)
	if err != nil {
		return err
	}
	var post model.ScheduledPost
	if err := s.store.DB.WithContext(ctx).First(&post, "id = ?", parsed).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || isMissingTableError(err) {
			return nil
		}
		return err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, post.ChatID); err != nil {
		return err
	}
	return s.store.DB.WithContext(ctx).Delete(&model.ScheduledPost{}, "id = ?", parsed).Error
}

func (s *postAPIService) Toggle(ctx context.Context, id string, ownerUserID string) (*api.Post, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	parsed, err := parseScheduledPostID(id)
	if err != nil {
		return nil, err
	}
	var post model.ScheduledPost
	if err := s.store.DB.WithContext(ctx).First(&post, "id = ?", parsed).Error; err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, post.ChatID); err != nil {
		return nil, err
	}
	post.Enabled = !post.Enabled
	if err := s.store.DB.WithContext(ctx).Model(&model.ScheduledPost{}).Where("id = ?", parsed).Update("enabled", post.Enabled).Error; err != nil {
		return nil, err
	}
	item := modelScheduledPostToAPI(post)
	return &item, nil
}

type scheduleAPIService struct{ store *store.Store }

func (s *scheduleAPIService) Create(ctx context.Context, req api.ScheduleCreateRequest) (*api.Schedule, error) {
	now := time.Now()
	job := api.Schedule{ID: uuid.NewString(), PostID: req.PostID, ChatID: req.ChatID, RunAt: req.RunAt, CronExpr: req.CronExpr, Timezone: req.Timezone, Enabled: req.Enabled, RepeatCount: req.RepeatCount, Note: req.Note, Status: "pending", CreatedAt: now, UpdatedAt: now}
	if s.store == nil || s.store.DB == nil {
		return &job, nil
	}
	runAt := req.RunAt
	scheduled := model.ScheduledJob{JobKey: job.ID, JobType: "publish", Status: "pending", TargetType: "post", RunAt: &runAt, PayloadJSON: "{}", MetadataJSON: fmt.Sprintf(`{"telegram_chat_id":%d}`, req.ChatID)}
	if err := s.store.DB.WithContext(ctx).Create(&scheduled).Error; err != nil {
		return nil, err
	}
	job.ID = scheduled.ID.String()
	return &job, nil
}

func (s *scheduleAPIService) List(ctx context.Context, query api.CommonListQuery) ([]api.Schedule, error) {
	if s.store == nil || s.store.DB == nil {
		return []api.Schedule{}, nil
	}
	var jobs []model.ScheduledJob
	if err := s.store.DB.WithContext(ctx).Order("created_at desc").Find(&jobs).Error; err != nil {
		return nil, err
	}
	ownedIDs, err := ownedTelegramChatIDs(ctx, s.store, query.OwnerUserID)
	if err != nil {
		return nil, err
	}
	out := make([]api.Schedule, 0, normalLimit(query.Limit))
	skipped := 0
	for _, job := range jobs {
		chatID := scheduleJobChatID(job)
		if !telegramChatIDInScope(chatID, 0, ownedIDs) {
			continue
		}
		if skipped < query.Offset {
			skipped++
			continue
		}
		out = append(out, modelScheduleToAPI(job))
		if len(out) >= normalLimit(query.Limit) {
			break
		}
	}
	return out, nil
}

func (s *scheduleAPIService) Get(ctx context.Context, id string, ownerUserID string) (*api.Schedule, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var job model.ScheduledJob
	if err := s.store.DB.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, scheduleJobChatID(job)); err != nil {
		return nil, err
	}
	item := modelScheduleToAPI(job)
	return &item, nil
}

func (s *scheduleAPIService) Update(ctx context.Context, id string, req api.ScheduleUpdateRequest, ownerUserID string) (*api.Schedule, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var job model.ScheduledJob
	if err := s.store.DB.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, scheduleJobChatID(job)); err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if req.RunAt != nil {
		updates["run_at"] = *req.RunAt
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(updates) > 0 {
		if err := s.store.DB.WithContext(ctx).Model(&model.ScheduledJob{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, id, ownerUserID)
}

func (s *scheduleAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	if s.store == nil || s.store.DB == nil {
		return nil
	}
	var job model.ScheduledJob
	if err := s.store.DB.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || isMissingTableError(err) {
			return nil
		}
		return err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, scheduleJobChatID(job)); err != nil {
		return err
	}
	return s.store.DB.WithContext(ctx).Delete(&model.ScheduledJob{}, "id = ?", id).Error
}

type statsAPIService struct{ store *store.Store }

func (s *statsAPIService) Overview(ctx context.Context, query api.StatsQuery) (*api.StatsOverview, error) {
	overview := &api.StatsOverview{}
	if s.store == nil || s.store.DB == nil {
		return overview, nil
	}
	db := s.store.DB.WithContext(ctx)
	ownedIDs, err := ownedTelegramChatIDs(ctx, s.store, query.OwnerUserID)
	if err != nil {
		return nil, err
	}
	chats := db.Model(&model.TelegramChat{})
	chats = scopeTelegramChatID(chats, "telegram_chat_id", query.ChatID, ownedIDs)
	_ = chats.Count(&overview.TotalChats).Error

	scheduledPosts := db.Model(&model.ScheduledPost{})
	scheduledPosts = scopeTelegramChatID(scheduledPosts, "chat_id", query.ChatID, ownedIDs)
	_ = scheduledPosts.Count(&overview.TotalPosts).Error

	jobCounts, err := scopedScheduledJobCounts(ctx, db, query.ChatID, ownedIDs)
	if err != nil {
		return nil, err
	}
	overview.OpenTasks = jobCounts.OpenTasks
	overview.TotalSchedules = jobCounts.TotalSchedules

	points := db.Model(&model.UserPoint{})
	points = scopeTelegramChatID(points, "chat_id", query.ChatID, ownedIDs)
	_ = points.Count(&overview.TotalMembers).Error

	from, to := statsRange(query)
	pointLogs := db.Model(&model.PointLog{}).Where("created_at >= ? AND created_at < ?", from, to)
	pointLogs = scopeTelegramChatID(pointLogs, "chat_id", query.ChatID, ownedIDs)
	_ = pointLogs.Distinct("user_id").Count(&overview.ActiveUsers).Error
	_ = pointLogs.Where("delta > 0").Select("COALESCE(SUM(delta), 0)").Scan(&overview.PointsIssued).Error
	return overview, nil
}

func (s *statsAPIService) Activity(ctx context.Context, query api.StatsQuery) ([]api.ActivityStats, error) {
	if s.store == nil || s.store.DB == nil {
		return []api.ActivityStats{}, nil
	}
	from, to := statsRange(query)
	ownedIDs, err := ownedTelegramChatIDs(ctx, s.store, query.OwnerUserID)
	if err != nil {
		return nil, err
	}
	db := s.store.DB.WithContext(ctx).Model(&model.PointLog{}).
		Where("created_at >= ? AND created_at < ?", from, to)
	db = scopeTelegramChatID(db, "chat_id", query.ChatID, ownedIDs)
	var logs []model.PointLog
	if err := db.Order("created_at asc").Find(&logs).Error; err != nil {
		if isMissingTableError(err) {
			return []api.ActivityStats{}, nil
		}
		return nil, err
	}
	byDate := map[string]*api.ActivityStats{}
	for day := beginningOfDay(from); day.Before(to); day = day.AddDate(0, 0, 1) {
		label := day.Format("2006-01-02")
		byDate[label] = &api.ActivityStats{Date: label}
	}
	for _, log := range logs {
		label := log.CreatedAt.Format("2006-01-02")
		item := byDate[label]
		if item == nil {
			item = &api.ActivityStats{Date: label}
			byDate[label] = item
		}
		if strings.HasPrefix(log.Reason, "command:") {
			item.Commands++
		} else {
			item.Messages++
		}
	}
	out := make([]api.ActivityStats, 0, len(byDate))
	for day := beginningOfDay(from); day.Before(to); day = day.AddDate(0, 0, 1) {
		label := day.Format("2006-01-02")
		out = append(out, *byDate[label])
	}
	return out, nil
}

func (s *statsAPIService) Points(ctx context.Context, query api.StatsQuery) ([]api.PointsStats, error) {
	if s.store == nil || s.store.DB == nil {
		return []api.PointsStats{}, nil
	}
	if query.From.IsZero() && query.To.IsZero() {
		db := s.store.DB.WithContext(ctx).Model(&model.UserPoint{})
		ownedIDs, err := ownedTelegramChatIDs(ctx, s.store, query.OwnerUserID)
		if err != nil {
			return nil, err
		}
		db = scopeTelegramChatID(db, "chat_id", query.ChatID, ownedIDs)
		var rows []model.UserPoint
		if err := db.Order("total_points desc").Limit(100).Find(&rows).Error; err != nil {
			if isMissingTableError(err) {
				return []api.PointsStats{}, nil
			}
			return nil, err
		}
		out := make([]api.PointsStats, 0, len(rows))
		for i, row := range rows {
			out = append(out, api.PointsStats{Rank: int64(i + 1), UserID: row.UserID, Points: row.TotalPoints})
		}
		return out, nil
	}
	from, to := statsRange(query)
	ownedIDs, err := ownedTelegramChatIDs(ctx, s.store, query.OwnerUserID)
	if err != nil {
		return nil, err
	}
	db := s.store.DB.WithContext(ctx).Model(&model.PointLog{}).
		Select("user_id, COALESCE(SUM(delta), 0) AS points").
		Where("created_at >= ? AND created_at < ?", from, to).
		Group("user_id").
		Order("points desc").
		Limit(100)
	db = scopeTelegramChatID(db, "chat_id", query.ChatID, ownedIDs)
	var rows []struct {
		UserID int64
		Points int64
	}
	if err := db.Scan(&rows).Error; err != nil {
		if isMissingTableError(err) {
			return []api.PointsStats{}, nil
		}
		return nil, err
	}
	out := make([]api.PointsStats, 0, len(rows))
	for i, row := range rows {
		out = append(out, api.PointsStats{Rank: int64(i + 1), UserID: row.UserID, Points: row.Points})
	}
	return out, nil
}

type userAPIService struct{ store *store.Store }

func (s *userAPIService) List(ctx context.Context, query api.UserListQuery) ([]api.UserRecord, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []api.UserRecord{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.UserPoint{})
	ownedIDs, err := ownedTelegramChatIDs(ctx, s.store, query.OwnerUserID)
	if err != nil {
		return nil, err
	}
	db = scopeTelegramChatID(db, "chat_id", query.ChatID, ownedIDs)
	var points []model.UserPoint
	if err := db.Order("updated_at desc").Limit(normalLimit(query.Limit)).Offset(query.Offset).Find(&points).Error; err != nil {
		if isMissingTableError(err) {
			return []api.UserRecord{}, nil
		}
		return nil, err
	}
	users, err := usersByTelegramID(ctx, s.store.DB, userPointIDs(points))
	if err != nil {
		return nil, err
	}
	out := make([]api.UserRecord, 0, len(points))
	for _, point := range points {
		item := api.UserRecord{
			ID:          point.UserID,
			Username:    fmt.Sprintf("user_%d", point.UserID),
			DisplayName: fmt.Sprintf("User %d", point.UserID),
			ChatID:      point.ChatID,
			TotalPoints: point.TotalPoints,
			Status:      "active",
			LastSeenAt:  point.UpdatedAt,
		}
		if user, ok := users[point.UserID]; ok {
			applyUserRecord(&item, user)
		}
		if matchesUserKeyword(item, query.Keyword) {
			out = append(out, item)
		}
	}
	return out, nil
}

func userPointIDs(points []model.UserPoint) []int64 {
	ids := make([]int64, 0, len(points))
	seen := make(map[int64]struct{}, len(points))
	for _, point := range points {
		if _, ok := seen[point.UserID]; ok {
			continue
		}
		seen[point.UserID] = struct{}{}
		ids = append(ids, point.UserID)
	}
	return ids
}

func usersByTelegramID(ctx context.Context, db *gorm.DB, ids []int64) (map[int64]model.User, error) {
	out := map[int64]model.User{}
	if db == nil || len(ids) == 0 {
		return out, nil
	}
	var users []model.User
	err := db.WithContext(ctx).Where("telegram_user_id IN ?", ids).Find(&users).Error
	if err != nil {
		if isMissingTableError(err) {
			return out, nil
		}
		return nil, err
	}
	for _, user := range users {
		if user.TelegramUserID == nil {
			continue
		}
		out[*user.TelegramUserID] = user
	}
	return out, nil
}

func applyUserRecord(item *api.UserRecord, user model.User) {
	if item == nil {
		return
	}
	if user.Username != nil && strings.TrimSpace(*user.Username) != "" {
		item.Username = *user.Username
	}
	if strings.TrimSpace(user.DisplayName) != "" {
		item.DisplayName = user.DisplayName
	}
	if strings.TrimSpace(user.Status) != "" {
		item.Status = normalizeUserStatus(user.Status)
	}
	if user.LastLoginAt != nil {
		item.LastSeenAt = *user.LastLoginAt
	} else {
		item.LastSeenAt = user.UpdatedAt
	}
}

type uuidCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeUUIDCursor(createdAt time.Time, id uuid.UUID) string {
	payload, err := json.Marshal(uuidCursor{CreatedAt: createdAt.UTC(), ID: id.String()})
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(payload)
}

func decodeUUIDCursor(cursor string) (time.Time, uuid.UUID, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return time.Time{}, uuid.UUID{}, fmt.Errorf("invalid cursor")
	}
	var payload uuidCursor
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return time.Time{}, uuid.UUID{}, fmt.Errorf("invalid cursor")
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(payload.ID))
	if err != nil || payload.CreatedAt.IsZero() {
		return time.Time{}, uuid.UUID{}, fmt.Errorf("invalid cursor")
	}
	return payload.CreatedAt, parsedID, nil
}

func decodePostCursor(cursor string) (time.Time, int64, error) {
	parts := strings.Split(strings.TrimSpace(cursor), "|")
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("invalid cursor")
	}
	cursorTime, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor")
	}
	cursorID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor")
	}
	return cursorTime, cursorID, nil
}

func EncodePostCursor(createdAt time.Time, id int64) string {
	return createdAt.UTC().Format(time.RFC3339Nano) + "|" + strconv.FormatInt(id, 10)
}
func normalLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func statsRange(query api.StatsQuery) (time.Time, time.Time) {
	now := time.Now()
	from := query.From
	to := query.To
	if from.IsZero() && to.IsZero() {
		to = now.AddDate(0, 0, 1)
		from = beginningOfDay(now).AddDate(0, 0, -6)
	} else if from.IsZero() {
		from = beginningOfDay(to).AddDate(0, 0, -6)
	} else if to.IsZero() {
		to = from.AddDate(0, 0, 7)
	} else {
		to = to.AddDate(0, 0, 1)
	}
	return beginningOfDay(from), beginningOfDay(to)
}

func beginningOfDay(value time.Time) time.Time {
	if value.IsZero() {
		value = time.Now()
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func normalizeUserStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "banned":
		return "banned"
	case "muted", "inactive", "disabled":
		return "muted"
	default:
		return "active"
	}
}

func matchesUserKeyword(item api.UserRecord, keyword string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return true
	}
	return strings.Contains(strings.ToLower(item.Username), keyword) ||
		strings.Contains(strings.ToLower(item.DisplayName), keyword) ||
		strings.Contains(strconv.FormatInt(item.ID, 10), keyword)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func timePtr(value time.Time) *time.Time { return &value }

type upsertTelegramOwnerInput struct {
	TelegramUserID int64
	Username       string
	DisplayName    string
	Role           string
	PhotoURL       string
}

func upsertTelegramOwner(ctx context.Context, st *store.Store, in upsertTelegramOwnerInput) (*model.User, error) {
	if st == nil || st.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	if in.TelegramUserID == 0 {
		return nil, errors.New("telegram user id is required")
	}
	role := strings.TrimSpace(in.Role)
	if role == "" {
		role = "owner"
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(in.Username)
	}
	now := time.Now()
	var user model.User
	err := st.DB.WithContext(ctx).Where("telegram_user_id = ?", in.TelegramUserID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		username := strings.TrimPrefix(strings.TrimSpace(in.Username), "@")
		user = model.User{
			BaseModel:      model.BaseModel{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
			TelegramUserID: &in.TelegramUserID,
			DisplayName:    displayName,
			Role:           role,
			LanguageCode:   "zh-CN",
			Timezone:       "Asia/Shanghai",
			Status:         "active",
			IsActive:       true,
			LastLoginAt:    &now,
		}
		if username != "" {
			user.Username = &username
		}
		if strings.TrimSpace(in.PhotoURL) != "" {
			user.MetadataJSON = fmt.Sprintf(`{"photo_url":%q}`, strings.TrimSpace(in.PhotoURL))
		}
		if user.MetadataJSON == "" {
			user.MetadataJSON = "{}"
		}
		if err := st.DB.WithContext(ctx).Create(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"display_name":  displayName,
		"last_login_at": now,
		"status":        "active",
		"is_active":     true,
		"updated_at":    now,
	}
	if strings.TrimSpace(user.Role) == "" || strings.TrimSpace(user.Role) == "user" {
		updates["role"] = role
		user.Role = role
	}
	if username := strings.TrimPrefix(strings.TrimSpace(in.Username), "@"); username != "" {
		updates["username"] = username
	}
	if err := st.DB.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := st.DB.WithContext(ctx).Where("telegram_user_id = ?", in.TelegramUserID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func ownedTelegramChatIDs(ctx context.Context, st *store.Store, ownerUserID string) ([]int64, error) {
	if strings.TrimSpace(ownerUserID) == "" {
		return nil, nil
	}
	if st == nil || st.DB == nil {
		return []int64{}, nil
	}
	ownerID, err := uuid.Parse(strings.TrimSpace(ownerUserID))
	if err != nil {
		return []int64{}, nil
	}
	var ids []int64
	err = st.DB.WithContext(ctx).
		Model(&model.TelegramChat{}).
		Where("owner_user_id = ?", ownerID).
		Pluck("telegram_chat_id", &ids).Error
	if isMissingTableError(err) {
		return []int64{}, nil
	}
	return ids, err
}

func ensureOwnedTelegramChatID(ctx context.Context, st *store.Store, ownerUserID string, chatID int64) error {
	if strings.TrimSpace(ownerUserID) == "" {
		return nil
	}
	if chatID == 0 {
		return api.ErrForbidden
	}
	ownedIDs, err := ownedTelegramChatIDs(ctx, st, ownerUserID)
	if err != nil {
		return err
	}
	if !containsTelegramChatID(ownedIDs, chatID) {
		return api.ErrForbidden
	}
	return nil
}

func ensureOwnedOptionalTelegramChatID(ctx context.Context, st *store.Store, ownerUserID string, chatID *int64) error {
	if strings.TrimSpace(ownerUserID) == "" {
		return nil
	}
	if chatID == nil {
		return api.ErrForbidden
	}
	return ensureOwnedTelegramChatID(ctx, st, ownerUserID, *chatID)
}

func queryTelegramChatIDs(ctx context.Context, st *store.Store, ownerUserID string, chatID int64) ([]int64, bool, error) {
	if strings.TrimSpace(ownerUserID) == "" {
		if chatID == 0 {
			return nil, false, nil
		}
		return []int64{chatID}, true, nil
	}
	ownedIDs, err := ownedTelegramChatIDs(ctx, st, ownerUserID)
	if err != nil {
		return nil, true, err
	}
	if chatID == 0 {
		return ownedIDs, true, nil
	}
	if containsTelegramChatID(ownedIDs, chatID) {
		return []int64{chatID}, true, nil
	}
	return []int64{}, true, nil
}

func containsTelegramChatID(ids []int64, chatID int64) bool {
	for _, id := range ids {
		if id == chatID {
			return true
		}
	}
	return false
}

func telegramChatIDInScope(chatID int64, requestedChatID int64, ownedIDs []int64) bool {
	if requestedChatID != 0 && chatID != requestedChatID {
		return false
	}
	if ownedIDs == nil {
		return true
	}
	if chatID == 0 {
		return false
	}
	return containsTelegramChatID(ownedIDs, chatID)
}

func lotteryChatID(ctx context.Context, st *store.Store, id int64) (int64, error) {
	if st == nil || st.DB == nil {
		return 0, nil
	}
	var record model.Lottery
	if err := st.DB.WithContext(ctx).Select("chat_id").First(&record, "id = ?", id).Error; err != nil {
		return 0, err
	}
	return record.ChatID, nil
}

func levelConfigChatID(ctx context.Context, st *store.Store, id string) (int64, error) {
	if st == nil || st.DB == nil {
		return 0, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return 0, err
	}
	var record model.LevelConfig
	if err := st.DB.WithContext(ctx).Select("chat_id").First(&record, "id = ?", parsed).Error; err != nil {
		return 0, err
	}
	return record.ChatID, nil
}

func keywordFilterChatID(ctx context.Context, st *store.Store, id string) (int64, error) {
	if st == nil || st.DB == nil {
		return 0, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return 0, err
	}
	var record model.KeywordFilter
	if err := st.DB.WithContext(ctx).Select("chat_id").First(&record, "id = ?", parsed).Error; err != nil {
		return 0, err
	}
	return record.ChatID, nil
}

func autoReplyChatID(ctx context.Context, st *store.Store, id string) (int64, error) {
	if st == nil || st.DB == nil {
		return 0, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return 0, err
	}
	var record model.AutoReply
	if err := st.DB.WithContext(ctx).Select("chat_id").First(&record, "id = ?", parsed).Error; err != nil {
		return 0, err
	}
	return record.ChatID, nil
}

func messageTemplateChatID(ctx context.Context, st *store.Store, id string) (*int64, error) {
	if st == nil || st.DB == nil {
		return nil, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	var record model.MessageTemplate
	if err := st.DB.WithContext(ctx).Select("chat_id").First(&record, "id = ?", parsed).Error; err != nil {
		return nil, err
	}
	return record.ChatID, nil
}

func inviteLinkChatID(ctx context.Context, st *store.Store, id string) (int64, error) {
	if st == nil || st.DB == nil {
		return 0, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return 0, err
	}
	var record model.InviteLink
	if err := st.DB.WithContext(ctx).Select("chat_id").First(&record, "id = ?", parsed).Error; err != nil {
		return 0, err
	}
	return record.ChatID, nil
}

func scheduleJobChatID(job model.ScheduledJob) int64 {
	var metadata struct {
		TelegramChatID int64 `json:"telegram_chat_id"`
		ChatID         int64 `json:"chat_id"`
	}
	if strings.TrimSpace(job.MetadataJSON) != "" {
		_ = json.Unmarshal([]byte(job.MetadataJSON), &metadata)
	}
	if metadata.TelegramChatID != 0 {
		return metadata.TelegramChatID
	}
	return metadata.ChatID
}

type scheduledJobCounts struct {
	OpenTasks      int64
	TotalSchedules int64
}

func scopedScheduledJobCounts(ctx context.Context, db *gorm.DB, requestedChatID int64, ownedIDs []int64) (scheduledJobCounts, error) {
	counts := scheduledJobCounts{}
	if db == nil {
		return counts, nil
	}
	var jobs []model.ScheduledJob
	if err := db.WithContext(ctx).
		Model(&model.ScheduledJob{}).
		Select("status", "metadata_json").
		Find(&jobs).Error; err != nil {
		if isMissingTableError(err) {
			return counts, nil
		}
		return counts, err
	}
	for _, job := range jobs {
		if !telegramChatIDInScope(scheduleJobChatID(job), requestedChatID, ownedIDs) {
			continue
		}
		counts.TotalSchedules++
		switch strings.ToLower(strings.TrimSpace(job.Status)) {
		case "pending", "running":
			counts.OpenTasks++
		}
	}
	return counts, nil
}

func scopeTelegramChatID(db *gorm.DB, column string, chatID int64, ids []int64) *gorm.DB {
	if chatID != 0 {
		if ids != nil && !containsTelegramChatID(ids, chatID) {
			return db.Where("1 = 0")
		}
		return db.Where(column+" = ?", chatID)
	}
	if ids != nil {
		if len(ids) == 0 {
			return db.Where("1 = 0")
		}
		return db.Where(column+" IN ?", ids)
	}
	return db
}

func chatBindingToAPI(chat model.TelegramChat, boundBy string) *api.ChatBinding {
	ownerUserID := ""
	if chat.OwnerUserID != nil {
		ownerUserID = chat.OwnerUserID.String()
	}
	return &api.ChatBinding{
		ChatID:      chat.TelegramChatID,
		ChatType:    chat.Type,
		Title:       deref(chat.Title),
		Username:    deref(chat.Username),
		InviteLink:  deref(chat.InviteLink),
		BoundBy:     boundBy,
		Description: deref(chat.Description),
		OwnerUserID: ownerUserID,
		BoundAt:     chat.CreatedAt,
	}
}

func modelPostToAPI(post model.Post, chatID int64) *api.Post {
	return &api.Post{ID: post.ID.String(), ChatID: chatID, Title: deref(post.Title), Content: post.ContentText, PublishAt: post.PublishAt, Status: post.Status, CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt}
}

func modelScheduledPostToAPI(post model.ScheduledPost) api.Post {
	return api.Post{
		ID:                strconv.FormatUint(post.ID, 10),
		ChatID:            post.ChatID,
		Title:             post.Title,
		Content:           post.Content,
		MediaURL:          post.MediaURL,
		MediaType:         post.MediaType,
		CronExpr:          post.CronExpr,
		RunOnceAt:         post.RunOnceAt,
		Enabled:           post.Enabled,
		LastRunAt:         post.LastRunAt,
		PublishAt:         post.RunOnceAt,
		Status:            scheduledPostStatus(post.Enabled),
		CreatedAt:         post.CreatedAt,
		UpdatedAt:         post.CreatedAt,
		PinAfterSend:      post.PinAfterSend,
		AutoDeleteSeconds: post.AutoDeleteSeconds,
	}
}

func levelConfigToAPI(record model.LevelConfig) api.Level {
	return api.Level{
		ID:          record.ID.String(),
		ChatID:      record.ChatID,
		Level:       record.Level,
		Name:        record.Label,
		MinPoints:   record.MinPoints,
		Badge:       record.Badge,
		Permissions: levelConfigPermissions(record),
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func levelConfigPermissions(record model.LevelConfig) []string {
	permissions := make([]string, 0, 2)
	if record.CanPostLink {
		permissions = append(permissions, "post_link")
	}
	if record.CanPostMedia {
		permissions = append(permissions, "post_media")
	}
	return permissions
}

func hasPermission(permissions []string, needle string, defaultValue bool) bool {
	if len(permissions) == 0 {
		return defaultValue
	}
	for _, permission := range permissions {
		if strings.EqualFold(strings.TrimSpace(permission), needle) {
			return true
		}
	}
	return false
}

func parseScheduledPostID(id string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(id), 10, 64)
}

func normalizeScheduledPostMediaType(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "", "text":
		return "text"
	case "photo", "video", "document":
		return strings.ToLower(strings.TrimSpace(mediaType))
	default:
		return strings.ToLower(strings.TrimSpace(mediaType))
	}
}

func validateScheduledPostCreate(req api.PostCreateRequest, runOnceAt *time.Time) error {
	if req.ChatID == 0 {
		return errors.New("chat_id is required")
	}
	mediaType := normalizeScheduledPostMediaType(req.MediaType)
	content := strings.TrimSpace(req.Content)
	title := strings.TrimSpace(req.Title)
	mediaURL := strings.TrimSpace(req.MediaURL)
	if mediaType == "text" && content == "" && title == "" {
		return errors.New("content is required for text scheduled posts")
	}
	if mediaType != "text" && mediaURL == "" {
		return fmt.Errorf("media_url is required for %s scheduled posts", mediaType)
	}
	if runOnceAt == nil && strings.TrimSpace(req.CronExpr) == "" {
		return errors.New("run_once_at or cron_expr is required")
	}
	if runOnceAt != nil && strings.TrimSpace(req.CronExpr) != "" {
		return errors.New("run_once_at and cron_expr cannot be used together")
	}
	if cronExpr := strings.TrimSpace(req.CronExpr); cronExpr != "" {
		return validateScheduleExpression(cronExpr)
	}
	return nil
}

func validateScheduleExpression(expr string) error {
	if duration, ok := parseEveryExpression(expr); ok {
		if duration < time.Second {
			return errors.New("@every interval must be at least 1s")
		}
		return nil
	}
	if _, err := robfigcron.ParseStandard(expr); err != nil {
		return fmt.Errorf("invalid cron_expr: %w", err)
	}
	return nil
}

func parseEveryExpression(expr string) (time.Duration, bool) {
	const prefix = "@every "
	text := strings.TrimSpace(expr)
	if !strings.HasPrefix(text, prefix) {
		return 0, false
	}
	duration, err := time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(text, prefix)))
	if err != nil || duration <= 0 {
		return 0, true
	}
	return duration, true
}

func scheduledPostStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func botChatPointConfigToAPI(cfg bot.ChatPointConfig) *api.ChatPointConfig {
	return &api.ChatPointConfig{
		ChatID:          cfg.ChatID,
		PointText:       cfg.PointText,
		PointPhoto:      cfg.PointPhoto,
		PointSticker:    cfg.PointSticker,
		PointVideo:      cfg.PointVideo,
		PointFile:       cfg.PointFile,
		PointVoice:      cfg.PointVoice,
		CooldownSeconds: cfg.CooldownSeconds,
		Enabled:         cfg.Enabled,
	}
}

func userPointToAPI(point model.UserPoint) *api.PointUserResponse {
	return &api.PointUserResponse{
		UserID:      point.UserID,
		ChatID:      point.ChatID,
		TotalPoints: point.TotalPoints,
		UpdatedAt:   point.UpdatedAt,
	}
}

func pointLogToAPI(log model.PointLog) api.PointLogItem {
	return api.PointLogItem{
		ID:        log.ID,
		UserID:    log.UserID,
		ChatID:    log.ChatID,
		Delta:     log.Delta,
		Reason:    log.Reason,
		CreatedAt: log.CreatedAt,
	}
}

func levelRuleToAPI(rule LevelRule) api.Level {
	return api.Level{
		ID:          strconv.Itoa(rule.Level),
		Level:       rule.Level,
		ChatID:      rule.ChatID,
		Name:        rule.Name,
		MinPoints:   rule.MinPoints,
		Badge:       rule.Badge,
		Permissions: levelPermissions(rule),
		CreatedAt:   rule.CreatedAt,
		UpdatedAt:   rule.UpdatedAt,
	}
}

func levelPermissions(rule LevelRule) []string {
	if rule.Level <= 1 {
		return []string{}
	}
	return []string{"post_link", "post_media"}
}

func violationRecordToAPI(record model.ViolationRecord) api.AdminViolation {
	status := "open"
	var resolvedAt *time.Time
	if record.Cleared {
		status = "resolved"
		resolvedAt = &record.UpdatedAt
	}
	return api.AdminViolation{
		ID:         record.ID.String(),
		ChatID:     record.ChatID,
		UserID:     record.UserID,
		Type:       record.ViolationType,
		Reason:     record.MessageText,
		Source:     record.DetectedBy,
		Status:     status,
		Resolution: record.ActionTaken,
		CreatedAt:  record.CreatedAt,
		ResolvedAt: resolvedAt,
	}
}

func keywordFilterToAPI(record model.KeywordFilter) api.Keyword {
	return api.Keyword{
		ID:        record.ID.String(),
		ChatID:    record.ChatID,
		Pattern:   record.Keyword,
		Keyword:   record.Keyword,
		MatchType: record.MatchType,
		Action:    record.Action,
		Scope:     record.Scope,
		ReplyText: record.ReplyText,
		Enabled:   record.Enabled,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func autoReplyRecordToAPI(record model.AutoReply) api.AutoReply {
	return api.AutoReply{
		ID:        record.ID.String(),
		ChatID:    record.ChatID,
		Keyword:   record.Keyword,
		MatchType: record.MatchType,
		ReplyText: record.ReplyText,
		Enabled:   record.Enabled,
		CreatedBy: record.CreatedBy,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func messageTemplateToAPI(record model.MessageTemplate) api.Template {
	return api.Template{
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
	}
}

type stringIDCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeStringIDCursor(createdAt time.Time, id string) string {
	payload, err := json.Marshal(stringIDCursor{CreatedAt: createdAt.UTC(), ID: id})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeStringIDCursor(cursor string) (time.Time, string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	var payload stringIDCursor
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	if payload.ID == "" || payload.CreatedAt.IsZero() {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	return payload.CreatedAt, payload.ID, nil
}
func inviteLinkToAPI(record model.InviteLink) api.InviteLink {
	return api.InviteLink{
		ID:                 record.ID.String(),
		ChatID:             record.ChatID,
		Name:               record.Name,
		InviteLink:         record.InviteLink,
		CreatesJoinRequest: record.CreatesJoinRequest,
		JoinCount:          record.JoinCount,
		CreatedBy:          record.CreatedBy,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}
}

func requestKeyword(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstStringPtr(values ...*string) *string {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func modelScheduleToAPI(job model.ScheduledJob) api.Schedule {
	runAt := time.Time{}
	if job.RunAt != nil {
		runAt = *job.RunAt
	}
	return api.Schedule{ID: job.ID.String(), ChatID: scheduleJobChatID(job), RunAt: runAt, Enabled: job.Status == "pending", Status: job.Status, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt}
}
