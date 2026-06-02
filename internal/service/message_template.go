package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

type MessageTemplateService struct {
	store *store.Store
}

func NewMessageTemplateService(st *store.Store) *MessageTemplateService {
	return &MessageTemplateService{store: st}
}

type MessageTemplateListFilter struct {
	ChatID int64
	Limit  int
	Offset int
	Cursor string
}

type MessageTemplateCreate struct {
	ChatID    *int64
	Name      string
	Content   string
	MediaType string
	MediaURL  string
	ParseMode string
	CreatedBy int64
}

type MessageTemplatePatch struct {
	ChatID    **int64
	Name      *string
	Content   *string
	MediaType *string
	MediaURL  *string
	ParseMode *string
}

func (s *MessageTemplateService) List(ctx context.Context, filter MessageTemplateListFilter) ([]model.MessageTemplate, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []model.MessageTemplate{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.MessageTemplate{})
	if filter.ChatID != 0 {
		db = db.Where("chat_id IS NULL OR chat_id = ?", filter.ChatID)
	}
	limit := normalLimit(filter.Limit)
	if strings.TrimSpace(filter.Cursor) != "" {
		cursorTime, cursorID, err := decodeUUIDCursor(filter.Cursor)
		if err != nil {
			return nil, err
		}
		db = db.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorTime, cursorTime, cursorID)
	}
	query := db.Order("chat_id NULLS FIRST, created_at desc, id desc").Limit(limit)
	if strings.TrimSpace(filter.Cursor) == "" {
		query = query.Offset(filter.Offset)
	}
	var records []model.MessageTemplate
	err := query.Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []model.MessageTemplate{}, nil
		}
		return nil, err
	}
	return records, nil
}

func (s *MessageTemplateService) Create(ctx context.Context, req MessageTemplateCreate) (model.MessageTemplate, error) {
	record := model.MessageTemplate{
		BaseModel: model.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		ChatID:    req.ChatID,
		Name:      strings.TrimSpace(req.Name),
		Content:   strings.TrimSpace(req.Content),
		MediaType: normalizeTemplateMediaType(req.MediaType),
		MediaURL:  strings.TrimSpace(req.MediaURL),
		ParseMode: normalizeTemplateParseMode(req.ParseMode),
		CreatedBy: req.CreatedBy,
	}
	if err := validateMessageTemplate(record); err != nil {
		return model.MessageTemplate{}, err
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return record, nil
	}
	if err := s.store.DB.WithContext(ctx).Select("*").Create(&record).Error; err != nil {
		if isMissingTableError(err) {
			return model.MessageTemplate{}, err
		}
		return model.MessageTemplate{}, err
	}
	return record, nil
}

func (s *MessageTemplateService) Update(ctx context.Context, id string, patch MessageTemplatePatch) (model.MessageTemplate, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return model.MessageTemplate{}, gorm.ErrInvalidDB
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return model.MessageTemplate{}, err
	}
	var record model.MessageTemplate
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		return model.MessageTemplate{}, err
	}
	updates := map[string]any{"updated_at": time.Now()}
	if patch.ChatID != nil {
		record.ChatID = *patch.ChatID
		updates["chat_id"] = *patch.ChatID
	}
	if patch.Name != nil {
		record.Name = strings.TrimSpace(*patch.Name)
		updates["name"] = record.Name
	}
	if patch.Content != nil {
		record.Content = strings.TrimSpace(*patch.Content)
		updates["content"] = record.Content
	}
	if patch.MediaType != nil {
		record.MediaType = normalizeTemplateMediaType(*patch.MediaType)
		updates["media_type"] = record.MediaType
	}
	if patch.MediaURL != nil {
		record.MediaURL = strings.TrimSpace(*patch.MediaURL)
		updates["media_url"] = record.MediaURL
	}
	if patch.ParseMode != nil {
		record.ParseMode = normalizeTemplateParseMode(*patch.ParseMode)
		updates["parse_mode"] = record.ParseMode
	}
	if err := validateMessageTemplate(record); err != nil {
		return model.MessageTemplate{}, err
	}
	if err := s.store.DB.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
		return model.MessageTemplate{}, err
	}
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		return model.MessageTemplate{}, err
	}
	return record, nil
}

func (s *MessageTemplateService) Delete(ctx context.Context, id string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	err = s.store.DB.WithContext(ctx).Delete(&model.MessageTemplate{}, "id = ?", parsed).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func validateMessageTemplate(record model.MessageTemplate) error {
	if strings.TrimSpace(record.Name) == "" {
		return errors.New("name is required")
	}
	if normalizeTemplateMediaType(record.MediaType) != "text" && strings.TrimSpace(record.MediaURL) == "" {
		return errors.New("media_url is required for media templates")
	}
	if strings.TrimSpace(record.Content) == "" && strings.TrimSpace(record.MediaURL) == "" {
		return errors.New("content or media_url is required")
	}
	return nil
}

func normalizeTemplateMediaType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "photo", "video":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "text"
	}
}

func normalizeTemplateParseMode(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "HTML":
		return "HTML"
	case "MARKDOWN", "MARKDOWNV2":
		return "Markdown"
	default:
		return ""
	}
}
