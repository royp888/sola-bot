package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

// ButtonTemplateService manages reusable inline-keyboard ("custom quick
// keyboard / buttons") templates that admins can attach to posts.
type ButtonTemplateService struct {
	store *store.Store
}

func NewButtonTemplateService(st *store.Store) *ButtonTemplateService {
	return &ButtonTemplateService{store: st}
}

type ButtonTemplateListFilter struct {
	OwnerUserID string
	Scope       string
	Keyword     string
	Limit       int
	Offset      int
	Cursor      string
}

type ButtonTemplateCreate struct {
	OwnerUserID        *uuid.UUID
	Name               string
	Scope              string
	InlineKeyboardJSON string
	Description        *string
	IsActive           bool
	MetadataJSON       string
}

type ButtonTemplatePatch struct {
	Name               *string
	Scope              *string
	InlineKeyboardJSON *string
	Description        *string
	IsActive           *bool
	MetadataJSON       *string
}

func (s *ButtonTemplateService) List(ctx context.Context, filter ButtonTemplateListFilter) ([]model.ButtonTemplate, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []model.ButtonTemplate{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.ButtonTemplate{})
	if owner := strings.TrimSpace(filter.OwnerUserID); owner != "" {
		ownerID, err := uuid.Parse(owner)
		if err != nil {
			return nil, err
		}
		db = db.Where("owner_user_id IS NULL OR owner_user_id = ?", ownerID)
	}
	if scope := strings.TrimSpace(filter.Scope); scope != "" {
		db = db.Where("scope = ?", scope)
	}
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		db = db.Where("name ILIKE ?", "%"+kw+"%")
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
	var records []model.ButtonTemplate
	if err := query.Find(&records).Error; err != nil {
		if isMissingTableError(err) {
			return []model.ButtonTemplate{}, nil
		}
		return nil, err
	}
	return records, nil
}

func (s *ButtonTemplateService) Get(ctx context.Context, id string) (model.ButtonTemplate, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return model.ButtonTemplate{}, gorm.ErrRecordNotFound
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return model.ButtonTemplate{}, err
	}
	var record model.ButtonTemplate
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		return model.ButtonTemplate{}, err
	}
	return record, nil
}

func (s *ButtonTemplateService) Create(ctx context.Context, req ButtonTemplateCreate) (model.ButtonTemplate, error) {
	record := model.ButtonTemplate{
		BaseModel:          model.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OwnerUserID:        req.OwnerUserID,
		Name:               strings.TrimSpace(req.Name),
		Scope:              normalizeButtonTemplateScope(req.Scope),
		InlineKeyboardJSON: normalizeInlineKeyboardJSON(req.InlineKeyboardJSON),
		Description:        trimOptionalString(req.Description),
		IsActive:           req.IsActive,
		MetadataJSON:       normalizeMetadataJSON(req.MetadataJSON),
	}
	if err := validateButtonTemplate(record); err != nil {
		return model.ButtonTemplate{}, err
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return record, nil
	}
	if err := s.store.DB.WithContext(ctx).Select("*").Create(&record).Error; err != nil {
		return model.ButtonTemplate{}, err
	}
	return record, nil
}

func (s *ButtonTemplateService) Update(ctx context.Context, id string, patch ButtonTemplatePatch) (model.ButtonTemplate, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return model.ButtonTemplate{}, gorm.ErrInvalidDB
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return model.ButtonTemplate{}, err
	}
	var record model.ButtonTemplate
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		return model.ButtonTemplate{}, err
	}
	updates := map[string]any{"updated_at": time.Now()}
	if patch.Name != nil {
		record.Name = strings.TrimSpace(*patch.Name)
		updates["name"] = record.Name
	}
	if patch.Scope != nil {
		record.Scope = normalizeButtonTemplateScope(*patch.Scope)
		updates["scope"] = record.Scope
	}
	if patch.InlineKeyboardJSON != nil {
		record.InlineKeyboardJSON = normalizeInlineKeyboardJSON(*patch.InlineKeyboardJSON)
		updates["inline_keyboard_json"] = record.InlineKeyboardJSON
	}
	if patch.Description != nil {
		record.Description = trimOptionalString(patch.Description)
		updates["description"] = record.Description
	}
	if patch.IsActive != nil {
		record.IsActive = *patch.IsActive
		updates["is_active"] = record.IsActive
	}
	if patch.MetadataJSON != nil {
		record.MetadataJSON = normalizeMetadataJSON(*patch.MetadataJSON)
		updates["metadata_json"] = record.MetadataJSON
	}
	if err := validateButtonTemplate(record); err != nil {
		return model.ButtonTemplate{}, err
	}
	if err := s.store.DB.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
		return model.ButtonTemplate{}, err
	}
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		return model.ButtonTemplate{}, err
	}
	return record, nil
}

func (s *ButtonTemplateService) Delete(ctx context.Context, id string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	err = s.store.DB.WithContext(ctx).Delete(&model.ButtonTemplate{}, "id = ?", parsed).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func validateButtonTemplate(record model.ButtonTemplate) error {
	if strings.TrimSpace(record.Name) == "" {
		return errors.New("name is required")
	}
	if err := ValidateInlineKeyboardJSON(record.InlineKeyboardJSON); err != nil {
		return err
	}
	return nil
}

// ValidateInlineKeyboardJSON validates a Telegram inline-keyboard definition:
// a JSON array of rows, each row an array of buttons, each button an object
// with a non-empty "text" (and optional "url" / "callback_data").
func ValidateInlineKeyboardJSON(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return nil
	}
	var rows [][]struct {
		Text         string `json:"text"`
		URL          string `json:"url"`
		CallbackData string `json:"callback_data"`
	}
	if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
		return errors.New("inline_keyboard_json must be a JSON array of button rows")
	}
	for _, row := range rows {
		for _, btn := range row {
			if strings.TrimSpace(btn.Text) == "" {
				return errors.New("each inline button requires a non-empty text")
			}
		}
	}
	return nil
}

func normalizeButtonTemplateScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "chat", "personal":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "global"
	}
}

func normalizeInlineKeyboardJSON(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "null" {
		return "[]"
	}
	return trimmed
}

func normalizeMetadataJSON(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
