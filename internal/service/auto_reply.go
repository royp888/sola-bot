package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

const (
	autoReplyCacheTTL    = 5 * time.Minute
	autoReplyMaxMatches  = 3
	autoReplyDefaultType = "contains"
)

type AutoReplyService struct {
	store *store.Store
}

func NewAutoReplyService(st *store.Store) *AutoReplyService {
	return &AutoReplyService{store: st}
}

type AutoReplyListFilter struct {
	ChatID  int64
	Enabled *bool
	Limit   int
	Offset  int
}

type AutoReplyPatch struct {
	Keyword   *string
	MatchType *string
	ReplyText *string
	Enabled   *bool
}

type autoReplyCacheItem struct {
	Keyword   string `json:"keyword"`
	MatchType string `json:"match_type"`
	ReplyText string `json:"reply_text"`
}

func autoReplyCacheKey(chatID int64) string {
	return fmt.Sprintf("auto_reply:%d", chatID)
}

func (s *AutoReplyService) MatchAll(ctx context.Context, chatID int64, text string) ([]bot.AutoReplyMatch, error) {
	text = strings.TrimSpace(text)
	if text == "" || chatID == 0 || s == nil || s.store == nil || s.store.DB == nil {
		return []bot.AutoReplyMatch{}, nil
	}
	items, err := s.loadCachedReplies(ctx, chatID)
	if err != nil {
		return nil, err
	}

	lower := strings.ToLower(text)
	matches := make([]bot.AutoReplyMatch, 0, autoReplyMaxMatches)
	for _, item := range items {
		if matchAutoReplyKeyword(lower, text, item) {
			matches = append(matches, bot.AutoReplyMatch{
				Keyword:   item.Keyword,
				ReplyText: item.ReplyText,
			})
			if len(matches) >= autoReplyMaxMatches {
				break
			}
		}
	}
	return matches, nil
}

func (s *AutoReplyService) List(ctx context.Context, filter AutoReplyListFilter) ([]model.AutoReply, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []model.AutoReply{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.AutoReply{})
	if filter.ChatID != 0 {
		db = db.Where("chat_id = ?", filter.ChatID)
	}
	if filter.Enabled != nil {
		db = db.Where("enabled = ?", *filter.Enabled)
	}
	var records []model.AutoReply
	err := db.Order("created_at desc").
		Limit(normalLimit(filter.Limit)).
		Offset(filter.Offset).
		Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []model.AutoReply{}, nil
		}
		return nil, err
	}
	return records, nil
}

func (s *AutoReplyService) ListForChat(ctx context.Context, chatID int64) ([]bot.AutoReplyRecord, error) {
	records, err := s.List(ctx, AutoReplyListFilter{ChatID: chatID, Limit: 100})
	if err != nil {
		return nil, err
	}
	out := make([]bot.AutoReplyRecord, 0, len(records))
	for _, record := range records {
		out = append(out, autoReplyToBot(record))
	}
	return out, nil
}

func (s *AutoReplyService) Create(ctx context.Context, record *model.AutoReply) (model.AutoReply, error) {
	if record == nil {
		return model.AutoReply{}, errors.New("auto reply is required")
	}
	record.Keyword = strings.TrimSpace(record.Keyword)
	record.ReplyText = strings.TrimSpace(record.ReplyText)
	record.MatchType = normalizeAutoReplyMatchType(record.MatchType)
	if err := validateAutoReply(*record); err != nil {
		return model.AutoReply{}, err
	}
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	if s == nil || s.store == nil || s.store.DB == nil {
		return *record, nil
	}
	if err := s.store.DB.WithContext(ctx).Select("*").Create(record).Error; err != nil {
		if isMissingTableError(err) {
			return model.AutoReply{}, err
		}
		return model.AutoReply{}, err
	}
	s.invalidateCache(ctx, record.ChatID)
	return *record, nil
}

func (s *AutoReplyService) CreateForBot(ctx context.Context, req bot.AutoReplyCreate) (bot.AutoReplyRecord, error) {
	record, err := s.Create(ctx, &model.AutoReply{
		ChatID:    req.ChatID,
		Keyword:   req.Keyword,
		MatchType: req.MatchType,
		ReplyText: req.ReplyText,
		Enabled:   req.Enabled,
		CreatedBy: req.CreatedBy,
	})
	if err != nil {
		return bot.AutoReplyRecord{}, err
	}
	return autoReplyToBot(record), nil
}

func (s *AutoReplyService) Update(ctx context.Context, id string, patch AutoReplyPatch) (model.AutoReply, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return model.AutoReply{}, gorm.ErrInvalidDB
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return model.AutoReply{}, err
	}
	var record model.AutoReply
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		if isMissingTableError(err) {
			return model.AutoReply{}, err
		}
		return model.AutoReply{}, err
	}

	updates := map[string]any{"updated_at": time.Now()}
	if patch.Keyword != nil {
		keyword := strings.TrimSpace(*patch.Keyword)
		if keyword == "" {
			return model.AutoReply{}, errors.New("keyword is required")
		}
		updates["keyword"] = keyword
		record.Keyword = keyword
	}
	if patch.MatchType != nil {
		matchType := normalizeAutoReplyMatchType(*patch.MatchType)
		updates["match_type"] = matchType
		record.MatchType = matchType
	}
	if patch.ReplyText != nil {
		replyText := strings.TrimSpace(*patch.ReplyText)
		if replyText == "" {
			return model.AutoReply{}, errors.New("reply_text is required")
		}
		updates["reply_text"] = replyText
		record.ReplyText = replyText
	}
	if patch.Enabled != nil {
		updates["enabled"] = *patch.Enabled
		record.Enabled = *patch.Enabled
	}
	if err := validateAutoReply(record); err != nil {
		return model.AutoReply{}, err
	}
	if err := s.store.DB.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
		if isMissingTableError(err) {
			return model.AutoReply{}, err
		}
		return model.AutoReply{}, err
	}
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		return model.AutoReply{}, err
	}
	s.invalidateCache(ctx, record.ChatID)
	return record, nil
}

func (s *AutoReplyService) Delete(ctx context.Context, id string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	var record model.AutoReply
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || isMissingTableError(err) {
			return nil
		}
		return err
	}
	if err := s.store.DB.WithContext(ctx).Delete(&record).Error; err != nil {
		if isMissingTableError(err) {
			return nil
		}
		return err
	}
	s.invalidateCache(ctx, record.ChatID)
	return nil
}

func (s *AutoReplyService) DeleteByKeyword(ctx context.Context, chatID int64, keyword string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return errors.New("keyword is required")
	}
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND keyword = ?", chatID, keyword).
		Delete(&model.AutoReply{}).Error
	if isMissingTableError(err) {
		return nil
	}
	if err != nil {
		return err
	}
	s.invalidateCache(ctx, chatID)
	return nil
}

func (s *AutoReplyService) loadCachedReplies(ctx context.Context, chatID int64) ([]autoReplyCacheItem, error) {
	key := autoReplyCacheKey(chatID)
	if s.store.Redis != nil {
		if raw, err := s.store.Redis.Get(ctx, key).Result(); err == nil && raw != "" {
			var cached []autoReplyCacheItem
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return cached, nil
			}
		}
	}

	var records []model.AutoReply
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND enabled = ?", chatID, true).
		Order("created_at asc").
		Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []autoReplyCacheItem{}, nil
		}
		return nil, err
	}

	items := make([]autoReplyCacheItem, 0, len(records))
	for _, record := range records {
		items = append(items, autoReplyCacheItem{
			Keyword:   record.Keyword,
			MatchType: record.MatchType,
			ReplyText: record.ReplyText,
		})
	}
	if s.store.Redis != nil {
		if data, err := json.Marshal(items); err == nil {
			_ = s.store.Redis.Set(ctx, key, data, autoReplyCacheTTL).Err()
		}
	}
	return items, nil
}

func (s *AutoReplyService) invalidateCache(ctx context.Context, chatID int64) {
	if s != nil && s.store != nil && s.store.Redis != nil {
		_ = s.store.Redis.Del(ctx, autoReplyCacheKey(chatID)).Err()
	}
}

func validateAutoReply(record model.AutoReply) error {
	if record.ChatID == 0 {
		return errors.New("chat_id is required")
	}
	if strings.TrimSpace(record.Keyword) == "" {
		return errors.New("keyword is required")
	}
	if strings.TrimSpace(record.ReplyText) == "" {
		return errors.New("reply_text is required")
	}
	if normalizeAutoReplyMatchType(record.MatchType) == "regex" {
		if _, err := regexp.Compile(record.Keyword); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}
	return nil
}

func normalizeAutoReplyMatchType(matchType string) string {
	switch strings.ToLower(strings.TrimSpace(matchType)) {
	case "exact", "regex":
		return strings.ToLower(strings.TrimSpace(matchType))
	default:
		return autoReplyDefaultType
	}
}

func matchAutoReplyKeyword(lowerText string, rawText string, item autoReplyCacheItem) bool {
	keyword := strings.TrimSpace(item.Keyword)
	if keyword == "" {
		return false
	}
	switch normalizeAutoReplyMatchType(item.MatchType) {
	case "exact":
		return strings.EqualFold(strings.TrimSpace(rawText), keyword)
	case "regex":
		matched, _ := regexp.MatchString(keyword, rawText)
		return matched
	default:
		return strings.Contains(lowerText, strings.ToLower(keyword))
	}
}

func autoReplyToBot(record model.AutoReply) bot.AutoReplyRecord {
	return bot.AutoReplyRecord{
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
