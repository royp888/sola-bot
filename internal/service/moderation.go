package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

type ModerationService struct {
	store *store.Store
}

func NewModerationService(st *store.Store) *ModerationService {
	return &ModerationService{store: st}
}

type ModerationConfigPatch struct {
	VerifyEnabled        *bool
	VerifyType           *string
	VerifyTimeoutSeconds *int
	VerifyQuestion       *string
	VerifyOptions        *string
	VerifyCorrectIndex   *int
	WarnLimit            *int
	BlockLinks           *bool
	LinkWhitelist        *string
	LinkBlacklist        *string
	BlockForwards        *bool
	BlockMedia           *bool
	KeywordFilterEnabled *bool
	SpamScoreThreshold   *int
	AiFilterEnabled      *bool
	RestrictUnverified   *bool
	WelcomeText          *string
	WelcomeDeleteSeconds *int
}

type KeywordFilterListFilter struct {
	ChatID  int64
	Scope   string
	Action  string
	Enabled *bool
	Limit   int
	Offset  int
}

type KeywordFilterCreate struct {
	ChatID    int64
	Keyword   string
	MatchType string
	Action    string
	Scope     string
	ReplyText string
	Enabled   *bool
	CreatedBy int64
}

type KeywordFilterPatch struct {
	Keyword   *string
	MatchType *string
	Action    *string
	Scope     *string
	ReplyText *string
	Enabled   *bool
}

type ViolationListFilter struct {
	ChatID  int64
	ChatIDs []int64
	UserID  int64
	Type    string
	Status  string
	Limit   int
	Offset  int
	Cursor  string
}

func (s *ModerationService) GetConfig(ctx context.Context, chatID int64) (model.ChatModerationConfig, error) {
	cfg := defaultModerationConfig(chatID)
	if s == nil || s.store == nil || s.store.DB == nil {
		return cfg, nil
	}
	db := s.store.DB.WithContext(ctx)
	err := db.Select("*").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chat_id"}},
		DoNothing: true,
	}).Create(&cfg).Error
	if err != nil && !isDuplicateKeyError(err) {
		if isMissingTableError(err) {
			return defaultModerationConfig(chatID), nil
		}
		return model.ChatModerationConfig{}, err
	}
	if err := db.First(&cfg, "chat_id = ?", chatID).Error; err != nil {
		if isMissingTableError(err) {
			return defaultModerationConfig(chatID), nil
		}
		return model.ChatModerationConfig{}, err
	}
	normalizeModerationConfig(&cfg)
	return cfg, nil
}

func (s *ModerationService) GetModerationConfig(ctx context.Context, chatID int64) (bot.ChatModerationConfig, error) {
	cfg, err := s.GetConfig(ctx, chatID)
	if err != nil {
		return bot.ChatModerationConfig{}, err
	}
	return bot.ChatModerationConfig{
		ChatID:               cfg.ChatID,
		BlockLinks:           cfg.BlockLinks,
		BlockForwards:        cfg.BlockForwards,
		BlockMedia:           cfg.BlockMedia,
		KeywordFilterEnabled: cfg.KeywordFilterEnabled,
		SpamScoreThreshold:   cfg.SpamScoreThreshold,
		AiFilterEnabled:      cfg.AiFilterEnabled,
		RestrictUnverified:   cfg.RestrictUnverified,
	}, nil
}

func (s *ModerationService) UpdateConfig(ctx context.Context, chatID int64, patch ModerationConfigPatch) (model.ChatModerationConfig, error) {
	cfg, err := s.GetConfig(ctx, chatID)
	if err != nil {
		return model.ChatModerationConfig{}, err
	}
	applyModerationConfigPatch(&cfg, patch)
	if s == nil || s.store == nil || s.store.DB == nil {
		return cfg, nil
	}
	err = s.store.DB.WithContext(ctx).
		Model(&model.ChatModerationConfig{}).
		Where("chat_id = ?", chatID).
		Updates(map[string]any{
			"verify_enabled":         cfg.VerifyEnabled,
			"verify_type":            cfg.VerifyType,
			"verify_timeout_seconds": cfg.VerifyTimeoutSeconds,
			"verify_question":        cfg.VerifyQuestion,
			"verify_options":         cfg.VerifyOptions,
			"verify_correct_index":   cfg.VerifyCorrectIndex,
			"warn_limit":             cfg.WarnLimit,
			"block_links":            cfg.BlockLinks,
			"link_whitelist":         cfg.LinkWhitelist,
			"link_blacklist":         cfg.LinkBlacklist,
			"block_forwards":         cfg.BlockForwards,
			"block_media":            cfg.BlockMedia,
			"keyword_filter_enabled": cfg.KeywordFilterEnabled,
			"spam_score_threshold":   cfg.SpamScoreThreshold,
			"ai_filter_enabled":      cfg.AiFilterEnabled,
			"restrict_unverified":    cfg.RestrictUnverified,
			"welcome_text":           cfg.WelcomeText,
			"welcome_delete_seconds": cfg.WelcomeDeleteSeconds,
			"updated_at":             cfg.UpdatedAt,
		}).Error
	if err != nil {
		if isMissingTableError(err) {
			return cfg, nil
		}
		return model.ChatModerationConfig{}, err
	}
	return cfg, nil
}

func (s *ModerationService) ListKeywords(ctx context.Context, chatID int64) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "当前未配置过滤关键词。", nil
	}
	var records []model.KeywordFilter
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND enabled = ?", chatID, true).
		Order("created_at desc").
		Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return "当前未配置过滤关键词。", nil
		}
		return "", err
	}
	if len(records) == 0 {
		return "当前未配置过滤关键词。", nil
	}
	var b strings.Builder
	b.WriteString("过滤关键词")
	for _, record := range records {
		fmt.Fprintf(&b, "\n- %s（%s/%s）", record.Keyword, record.MatchType, record.Action)
	}
	return b.String(), nil
}

func (s *ModerationService) AddKeyword(ctx context.Context, chatID int64, keyword string, operatorID int64) (string, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return "", fmt.Errorf("keyword is required")
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return "关键词已添加。", nil
	}
	record := model.KeywordFilter{
		BaseModel: model.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		ChatID:    chatID,
		Keyword:   keyword,
		MatchType: "contains",
		Action:    "delete",
		Scope:     "chat",
		Enabled:   true,
		CreatedBy: operatorID,
	}
	err := s.store.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chat_id"}, {Name: "keyword"}},
		DoUpdates: clause.AssignmentColumns([]string{"match_type", "action", "scope", "enabled", "created_by", "updated_at"}),
	}).Create(&record).Error
	if err != nil {
		if isMissingTableError(err) {
			return "关键词已添加。", nil
		}
		return "", err
	}
	return "关键词已添加：" + keyword, nil
}

func (s *ModerationService) DeleteKeyword(ctx context.Context, chatID int64, keyword string, operatorID int64) (string, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return "", fmt.Errorf("keyword is required")
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return "关键词已删除。", nil
	}
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND keyword = ?", chatID, keyword).
		Delete(&model.KeywordFilter{}).Error
	if err != nil {
		if isMissingTableError(err) {
			return "关键词已删除。", nil
		}
		return "", err
	}
	return "关键词已删除：" + keyword, nil
}

func (s *ModerationService) ListKeywordFilters(ctx context.Context, filter KeywordFilterListFilter) ([]model.KeywordFilter, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []model.KeywordFilter{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.KeywordFilter{})
	if filter.ChatID != 0 {
		db = db.Where("chat_id = ?", filter.ChatID)
	}
	if strings.TrimSpace(filter.Scope) != "" {
		db = db.Where("scope = ?", strings.TrimSpace(filter.Scope))
	}
	if strings.TrimSpace(filter.Action) != "" {
		db = db.Where("action = ?", normalizeKeywordAction(filter.Action))
	}
	if filter.Enabled != nil {
		db = db.Where("enabled = ?", *filter.Enabled)
	}
	var records []model.KeywordFilter
	err := db.Order("created_at desc").
		Limit(normalLimit(filter.Limit)).
		Offset(filter.Offset).
		Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []model.KeywordFilter{}, nil
		}
		return nil, err
	}
	return records, nil
}

func (s *ModerationService) CreateKeywordFilter(ctx context.Context, req KeywordFilterCreate) (model.KeywordFilter, error) {
	keyword := strings.TrimSpace(req.Keyword)
	if keyword == "" {
		return model.KeywordFilter{}, fmt.Errorf("keyword is required")
	}
	now := time.Now()
	record := model.KeywordFilter{
		BaseModel: model.BaseModel{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
		ChatID:    req.ChatID,
		Keyword:   keyword,
		MatchType: normalizeKeywordMatchType(req.MatchType),
		Action:    normalizeKeywordAction(req.Action),
		Scope:     normalizeKeywordScope(req.Scope),
		ReplyText: strings.TrimSpace(req.ReplyText),
		Enabled:   true,
		CreatedBy: req.CreatedBy,
	}
	if req.Enabled != nil {
		record.Enabled = *req.Enabled
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return record, nil
	}
	err := s.store.DB.WithContext(ctx).Create(&record).Error
	if err != nil {
		if isMissingTableError(err) {
			return model.KeywordFilter{}, err
		}
		return model.KeywordFilter{}, err
	}
	return record, nil
}

func (s *ModerationService) UpdateKeywordFilter(ctx context.Context, id string, patch KeywordFilterPatch) (model.KeywordFilter, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return model.KeywordFilter{}, gorm.ErrInvalidDB
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return model.KeywordFilter{}, err
	}
	var record model.KeywordFilter
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		if isMissingTableError(err) {
			return model.KeywordFilter{}, err
		}
		return model.KeywordFilter{}, err
	}
	updates := map[string]any{"updated_at": time.Now()}
	if patch.Keyword != nil {
		keyword := strings.TrimSpace(*patch.Keyword)
		if keyword == "" {
			return model.KeywordFilter{}, fmt.Errorf("keyword is required")
		}
		updates["keyword"] = keyword
	}
	if patch.MatchType != nil {
		updates["match_type"] = normalizeKeywordMatchType(*patch.MatchType)
	}
	if patch.Action != nil {
		updates["action"] = normalizeKeywordAction(*patch.Action)
	}
	if patch.Scope != nil {
		updates["scope"] = normalizeKeywordScope(*patch.Scope)
	}
	if patch.ReplyText != nil {
		updates["reply_text"] = strings.TrimSpace(*patch.ReplyText)
	}
	if patch.Enabled != nil {
		updates["enabled"] = *patch.Enabled
	}
	if err := s.store.DB.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
		if isMissingTableError(err) {
			return model.KeywordFilter{}, err
		}
		return model.KeywordFilter{}, err
	}
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		return model.KeywordFilter{}, err
	}
	return record, nil
}

func (s *ModerationService) DeleteKeywordFilter(ctx context.Context, id string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	err = s.store.DB.WithContext(ctx).Delete(&model.KeywordFilter{}, "id = ?", parsed).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func (s *ModerationService) MatchKeyword(ctx context.Context, chatID int64, text string) (bot.KeywordFilterMatch, error) {
	text = strings.TrimSpace(text)
	if text == "" || s == nil || s.store == nil || s.store.DB == nil {
		return bot.KeywordFilterMatch{}, nil
	}

	var records []model.KeywordFilter
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND enabled = ?", chatID, true).
		Order("created_at desc").
		Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return bot.KeywordFilterMatch{}, nil
		}
		return bot.KeywordFilterMatch{}, err
	}

	for _, record := range records {
		if keywordMatches(text, record.Keyword, record.MatchType) {
			action := normalizeKeywordAction(record.Action)
			return bot.KeywordFilterMatch{
				Matched:   true,
				Keyword:   record.Keyword,
				MatchType: normalizeKeywordMatchType(record.MatchType),
				Action:    action,
				ReplyText: record.ReplyText,
			}, nil
		}
	}
	return bot.KeywordFilterMatch{}, nil
}

func (s *ModerationService) RecordKeywordViolation(ctx context.Context, violation bot.KeywordViolation) error {
	return s.RecordViolation(ctx, model.ViolationRecord{
		UserID:          violation.UserID,
		ChatID:          violation.ChatID,
		ViolationType:   violation.ViolationType,
		ActionTaken:     violation.ActionTaken,
		MessageText:     violation.MessageText,
		DetectedBy:      violation.DetectedBy,
		DurationSeconds: violation.DurationSeconds,
	})
}

func (s *ModerationService) RecordViolation(ctx context.Context, record model.ViolationRecord) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	if record.DetectedBy == "" {
		record.DetectedBy = "rule"
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	err := s.store.DB.WithContext(ctx).Create(&record).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func keywordMatches(text string, keyword string, matchType string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	matchType = normalizeKeywordMatchType(matchType)
	lowerText := strings.ToLower(text)
	lowerKeyword := strings.ToLower(keyword)
	switch matchType {
	case "exact":
		return lowerText == lowerKeyword
	case "prefix":
		return strings.HasPrefix(lowerText, lowerKeyword)
	case "suffix":
		return strings.HasSuffix(lowerText, lowerKeyword)
	case "regex":
		re, err := regexp.Compile(keyword)
		return err == nil && re.MatchString(text)
	default:
		return strings.Contains(lowerText, lowerKeyword)
	}
}

func normalizeKeywordMatchType(matchType string) string {
	switch strings.ToLower(strings.TrimSpace(matchType)) {
	case "exact", "prefix", "suffix", "regex":
		return strings.ToLower(strings.TrimSpace(matchType))
	default:
		return "contains"
	}
}

func normalizeKeywordAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "warn", "mute", "ban", "record", "ignore":
		return strings.ToLower(strings.TrimSpace(action))
	case "delete", "delete_message", "remove":
		return "delete_message"
	default:
		return "delete_message"
	}
}

func normalizeKeywordScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return "chat"
	}
	return scope
}

func (s *ModerationService) ListViolations(ctx context.Context, chatID int64, userID int64, limit int, offset int) ([]model.ViolationRecord, error) {
	return s.ListViolationsFiltered(ctx, ViolationListFilter{ChatID: chatID, UserID: userID, Limit: limit, Offset: offset})
}

func (s *ModerationService) ListViolationsFiltered(ctx context.Context, filter ViolationListFilter) ([]model.ViolationRecord, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []model.ViolationRecord{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.ViolationRecord{})
	if filter.ChatID != 0 {
		db = db.Where("chat_id = ?", filter.ChatID)
	} else if filter.ChatIDs != nil {
		if len(filter.ChatIDs) == 0 {
			return []model.ViolationRecord{}, nil
		}
		db = db.Where("chat_id IN ?", filter.ChatIDs)
	}
	if filter.UserID != 0 {
		db = db.Where("user_id = ?", filter.UserID)
	}
	if strings.TrimSpace(filter.Type) != "" {
		db = db.Where("violation_type = ?", strings.TrimSpace(filter.Type))
	}
	switch normalizeViolationStatus(filter.Status) {
	case "open":
		db = db.Where("cleared = ?", false)
	case "resolved":
		db = db.Where("cleared = ?", true)
	}
	limit := normalLimit(filter.Limit)
	if strings.TrimSpace(filter.Cursor) != "" {
		cursorTime, cursorID, err := decodeUUIDCursor(filter.Cursor)
		if err != nil {
			return nil, err
		}
		db = db.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorTime, cursorTime, cursorID)
	}
	query := db.Order("created_at desc, id desc").Limit(limit)
	if strings.TrimSpace(filter.Cursor) == "" {
		query = query.Offset(filter.Offset)
	}
	var records []model.ViolationRecord
	err := query.Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []model.ViolationRecord{}, nil
		}
		return nil, err
	}
	return records, nil
}

func (s *ModerationService) UpdateViolation(ctx context.Context, id string, status *string, resolution *string) (model.ViolationRecord, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return model.ViolationRecord{}, gorm.ErrInvalidDB
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return model.ViolationRecord{}, err
	}
	var record model.ViolationRecord
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		if isMissingTableError(err) {
			return model.ViolationRecord{}, err
		}
		return model.ViolationRecord{}, err
	}
	updates := map[string]any{"updated_at": time.Now()}
	if status != nil {
		updates["cleared"] = normalizeViolationStatus(*status) == "resolved"
	}
	if resolution != nil {
		updates["action_taken"] = strings.TrimSpace(*resolution)
	}
	if err := s.store.DB.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
		if isMissingTableError(err) {
			return model.ViolationRecord{}, err
		}
		return model.ViolationRecord{}, err
	}
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		return model.ViolationRecord{}, err
	}
	return record, nil
}

func normalizeViolationStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved", "cleared", "closed", "dismissed":
		return "resolved"
	case "open", "pending", "":
		return "open"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func (s *LevelService) SetLevel(ctx context.Context, chatID int64, userID int64, level int, operatorID int64) (string, error) {
	if level < 0 {
		return "", fmt.Errorf("level must be >= 0")
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return fmt.Sprintf("已将用户 %d 的等级设置为 %d。", userID, level), nil
	}
	name := defaultLevelName(level)
	rule, err := s.UpsertRule(ctx, LevelRule{
		ChatID:    chatID,
		Level:     level,
		Name:      name,
		MinPoints: 0,
		Badge:     "Lv." + intString(level),
		Enabled:   true,
	})
	if err != nil && !isMissingTableError(err) && err != gorm.ErrRecordNotFound {
		return "", err
	}
	return fmt.Sprintf("等级规则已保存：Lv.%d %s。用户 %d 当前会按积分匹配等级。", level, rule.Name, userID), nil
}

func defaultModerationConfig(chatID int64) model.ChatModerationConfig {
	return model.ChatModerationConfig{
		ChatID:               chatID,
		VerifyEnabled:        true,
		VerifyType:           "button",
		VerifyTimeoutSeconds: 60,
		VerifyQuestion:       "",
		VerifyOptions:        "[]",
		VerifyCorrectIndex:   -1,
		WarnLimit:            3,
		KeywordFilterEnabled: true,
		SpamScoreThreshold:   60,
		AiFilterEnabled:      false,
		RestrictUnverified:   true,
		WelcomeText:          "欢迎 {name}！",
		WelcomeDeleteSeconds: 30,
		UpdatedAt:            time.Now(),
	}
}

func normalizeModerationConfig(cfg *model.ChatModerationConfig) {
	if strings.TrimSpace(cfg.VerifyType) == "" {
		cfg.VerifyType = "button"
	}
	if cfg.VerifyTimeoutSeconds <= 0 {
		cfg.VerifyTimeoutSeconds = 60
	}
	if cfg.WarnLimit <= 0 {
		cfg.WarnLimit = 3
	}
	if cfg.SpamScoreThreshold <= 0 {
		cfg.SpamScoreThreshold = 60
	}
	if strings.TrimSpace(cfg.WelcomeText) == "" {
		cfg.WelcomeText = "欢迎 {name}！"
	}
	if cfg.WelcomeDeleteSeconds < 0 {
		cfg.WelcomeDeleteSeconds = 0
	}
	if cfg.VerifyOptions == "" {
		cfg.VerifyOptions = "[]"
	}
	if cfg.VerifyCorrectIndex < 0 {
		cfg.VerifyCorrectIndex = -1
	}
}

func parseDomainList(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var domains []string
	for _, line := range lines {
		d := strings.TrimSpace(line)
		if d != "" {
			domains = append(domains, strings.ToLower(d))
		}
	}
	return domains
}

func normalizeDomainList(text string) string {
	domains := parseDomainList(text)
	if len(domains) == 0 {
		return ""
	}
	return strings.Join(domains, "\n")
}

func applyModerationConfigPatch(cfg *model.ChatModerationConfig, patch ModerationConfigPatch) {
	if patch.VerifyEnabled != nil {
		cfg.VerifyEnabled = *patch.VerifyEnabled
	}
	if patch.VerifyType != nil {
		cfg.VerifyType = strings.TrimSpace(*patch.VerifyType)
	}
	if patch.VerifyTimeoutSeconds != nil {
		cfg.VerifyTimeoutSeconds = nonNegative(*patch.VerifyTimeoutSeconds)
	}
	if patch.VerifyQuestion != nil {
		cfg.VerifyQuestion = strings.TrimSpace(*patch.VerifyQuestion)
	}
	if patch.VerifyOptions != nil {
		cfg.VerifyOptions = strings.TrimSpace(*patch.VerifyOptions)
	}
	if patch.VerifyCorrectIndex != nil {
		cfg.VerifyCorrectIndex = *patch.VerifyCorrectIndex
	}
	if patch.WarnLimit != nil && *patch.WarnLimit > 0 {
		cfg.WarnLimit = *patch.WarnLimit
	}
	if patch.BlockLinks != nil {
		cfg.BlockLinks = *patch.BlockLinks
	}
	if patch.LinkWhitelist != nil {
		cfg.LinkWhitelist = normalizeDomainList(*patch.LinkWhitelist)
	}
	if patch.LinkBlacklist != nil {
		cfg.LinkBlacklist = normalizeDomainList(*patch.LinkBlacklist)
	}
	if patch.BlockForwards != nil {
		cfg.BlockForwards = *patch.BlockForwards
	}
	if patch.BlockMedia != nil {
		cfg.BlockMedia = *patch.BlockMedia
	}
	if patch.KeywordFilterEnabled != nil {
		cfg.KeywordFilterEnabled = *patch.KeywordFilterEnabled
	}
	if patch.SpamScoreThreshold != nil {
		cfg.SpamScoreThreshold = nonNegative(*patch.SpamScoreThreshold)
	}
	if patch.RestrictUnverified != nil {
		cfg.RestrictUnverified = *patch.RestrictUnverified
	}
	if patch.WelcomeText != nil {
		cfg.WelcomeText = strings.TrimSpace(*patch.WelcomeText)
	}
	if patch.WelcomeDeleteSeconds != nil {
		cfg.WelcomeDeleteSeconds = nonNegative(*patch.WelcomeDeleteSeconds)
	}
	cfg.UpdatedAt = time.Now()
	normalizeModerationConfig(cfg)
}
