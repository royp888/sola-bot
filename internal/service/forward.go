package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

// forwardAPIService backs the 自动转发 screens in the admin UI.
// It owns CRUD + 预览/试发; 实际搬运在 ForwardEngine（bot 进程实时、worker 进程历史）。
type forwardAPIService struct {
	store  *store.Store
	engine *ForwardEngine
}

func NewForwardAPIService(st *store.Store, botToken string) api.ForwardService {
	return &forwardAPIService{
		store:  st,
		engine: NewForwardEngine(st, botToken, nil),
	}
}

func (s *forwardAPIService) Create(ctx context.Context, req api.AutoForwardCreateRequest, ownerUserID string) (*api.AutoForward, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, errors.New("forward storage is unavailable")
	}
	if req.SourceChatID == 0 {
		return nil, errors.New("source_chat_id is required")
	}
	if req.TargetChatID == 0 {
		return nil, errors.New("target_chat_id is required")
	}
	if req.SourceChatID == req.TargetChatID {
		return nil, errors.New("source_chat_id and target_chat_id must be different")
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, req.SourceChatID); err != nil {
		return nil, err
	}
	if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, req.TargetChatID); err != nil {
		return nil, err
	}

	mode := normalizeForwardMode(req.Mode)
	now := time.Now()
	record := model.AutoForward{
		Name:                  strings.TrimSpace(req.Name),
		Enabled:               req.Enabled,
		SourceChatID:          req.SourceChatID,
		SourceTitle:           strings.TrimSpace(req.SourceTitle),
		TargetChatID:          req.TargetChatID,
		TargetTitle:           strings.TrimSpace(req.TargetTitle),
		Mode:                  mode,
		ParseMode:             normalizeForwardParseMode(req.ParseMode),
		Realtime:              true,
		HistoryState:          model.ForwardStateIdle,
		HistoryAutoStopMisses: 25,
		DelayMS:               800,
		BatchSize:             20,
		SkipDuplicates:        true,
		KeepOriginalText:      true,
		BuyText:               "🛒 立即购买",
		BuyButtonOnMedia:      true,
		BuyButtonOnText:       true,
		OwnerUserID:           parseUUIDPtr(ownerUserID),
		CreatedByUserID:       parseUUIDPtr(ownerUserID),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	applyForwardRequest(&record, req)

	if record.HistoryFromID <= 0 {
		record.HistoryFromID = 0
	}
	record.HistoryCursor = record.HistoryFromID
	if req.StartHistory {
		if record.HistoryFromID <= 0 {
			return nil, errors.New("start_history requires history_from_id")
		}
		record.HistoryState = model.ForwardStateRunning
		record.HistoryStartedAt = &now
	}
	if record.Enabled && record.Realtime && record.SourceChatID == 0 {
		return nil, errors.New("realtime forwarding requires source_chat_id")
	}

	if err := s.store.DB.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	out := toAPIAutoForward(record)
	return &out, nil
}

func (s *forwardAPIService) List(ctx context.Context, query api.CommonListQuery, ownerUserID string) ([]api.AutoForward, error) {
	if s.store == nil || s.store.DB == nil {
		return []api.AutoForward{}, nil
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	db := s.store.DB.WithContext(ctx).Model(&model.AutoForward{})
	if ownerUserID = strings.TrimSpace(ownerUserID); ownerUserID != "" {
		ownerID := parseUUIDPtr(ownerUserID)
		if ownerID == nil {
			return []api.AutoForward{}, nil
		}
		db = db.Where("owner_user_id = ?", *ownerID)
	}

	var records []model.AutoForward
	if err := db.Order("id desc").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		if isMissingTableError(err) {
			return []api.AutoForward{}, nil
		}
		return nil, err
	}
	out := make([]api.AutoForward, 0, len(records))
	for _, record := range records {
		out = append(out, toAPIAutoForward(record))
	}
	return out, nil
}

func (s *forwardAPIService) Get(ctx context.Context, id string, ownerUserID string) (*api.AutoForward, error) {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	out := toAPIAutoForward(*record)
	return &out, nil
}

func (s *forwardAPIService) Update(ctx context.Context, id string, req api.AutoForwardUpdateRequest, ownerUserID string) (*api.AutoForward, error) {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	if req.SourceChatID != 0 && req.SourceChatID != record.SourceChatID {
		if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, req.SourceChatID); err != nil {
			return nil, err
		}
	}
	if req.TargetChatID != 0 && req.TargetChatID != record.TargetChatID {
		if err := ensureOwnedTelegramChatID(ctx, s.store, ownerUserID, req.TargetChatID); err != nil {
			return nil, err
		}
	}
	applyForwardRequest(record, req)
	if record.SourceChatID == record.TargetChatID {
		return nil, errors.New("source_chat_id and target_chat_id must be different")
	}
	record.UpdatedAt = time.Now()
	if err := s.store.DB.WithContext(ctx).Save(record).Error; err != nil {
		return nil, err
	}
	out := toAPIAutoForward(*record)
	return &out, nil
}

func (s *forwardAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return err
	}
	if err := s.store.DB.WithContext(ctx).Delete(&model.AutoForward{}, "id = ?", record.ID).Error; err != nil {
		return err
	}
	err = s.store.DB.WithContext(ctx).Delete(&model.AutoForwardLog{}, "forward_id = ?", record.ID).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func (s *forwardAPIService) StartHistory(ctx context.Context, id string, ownerUserID string) (*api.AutoForward, error) {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	cursor := record.HistoryCursor
	if cursor <= 0 {
		cursor = record.HistoryFromID
	}
	if cursor <= 0 {
		return nil, errors.New("history_from_id is required before starting history forwarding")
	}
	now := time.Now()
	if err := s.store.DB.WithContext(ctx).Model(&model.AutoForward{}).Where("id = ?", record.ID).Updates(map[string]any{
		"history_state":       model.ForwardStateRunning,
		"history_cursor":      cursor,
		"history_started_at":  now,
		"history_finished_at": nil,
		"last_error":          "",
		"updated_at":          now,
	}).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, id, ownerUserID)
}

func (s *forwardAPIService) PauseHistory(ctx context.Context, id string, ownerUserID string) (*api.AutoForward, error) {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	if err := s.store.DB.WithContext(ctx).Model(&model.AutoForward{}).Where("id = ?", record.ID).Updates(map[string]any{
		"history_state": model.ForwardStatePaused,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, id, ownerUserID)
}

func (s *forwardAPIService) ResetHistory(ctx context.Context, id string, ownerUserID string) (*api.AutoForward, error) {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	if err := s.store.DB.WithContext(ctx).Model(&model.AutoForward{}).Where("id = ?", record.ID).Updates(map[string]any{
		"history_state":       model.ForwardStateIdle,
		"history_cursor":      record.HistoryFromID,
		"history_sent":        0,
		"history_failed":      0,
		"history_skipped":     0,
		"history_started_at":  nil,
		"history_finished_at": nil,
		"last_error":          "",
		"updated_at":          time.Now(),
	}).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, id, ownerUserID)
}

func (s *forwardAPIService) Logs(ctx context.Context, id string, query api.CommonListQuery, ownerUserID string) ([]api.AutoForwardLog, error) {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	var records []model.AutoForwardLog
	err = s.store.DB.WithContext(ctx).
		Where("forward_id = ?", record.ID).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []api.AutoForwardLog{}, nil
		}
		return nil, err
	}
	out := make([]api.AutoForwardLog, 0, len(records))
	for _, item := range records {
		out = append(out, api.AutoForwardLog{
			ID:              item.ID,
			ForwardID:       item.ForwardID,
			RuleName:        item.RuleName,
			SourceChatID:    item.SourceChatID,
			SourceMessageID: item.SourceMessageID,
			TargetChatID:    item.TargetChatID,
			TargetMessageID: item.TargetMessageID,
			Status:          item.Status,
			MediaType:       item.MediaType,
			FileSize:        item.FileSize,
			Text:            item.Text,
			Error:           item.Error,
			CreatedAt:       item.CreatedAt,
		})
	}
	return out, nil
}

func (s *forwardAPIService) Preview(ctx context.Context, id string, req api.AutoForwardPreviewRequest, ownerUserID string) (*api.AutoForwardPreviewResponse, error) {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	resp := RenderForwardPreview(*record, req)
	return &resp, nil
}

func (s *forwardAPIService) TestSend(ctx context.Context, id string, req api.AutoForwardPreviewRequest, ownerUserID string) (*api.AutoForwardLog, error) {
	record, err := s.findForward(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	entry, err := s.engine.TestSend(ctx, *record, req)
	if err != nil {
		return nil, err
	}
	out := api.AutoForwardLog{
		ID:              entry.ID,
		ForwardID:       entry.ForwardID,
		RuleName:        entry.RuleName,
		SourceChatID:    entry.SourceChatID,
		SourceMessageID: entry.SourceMessageID,
		TargetChatID:    entry.TargetChatID,
		TargetMessageID: entry.TargetMessageID,
		Status:          entry.Status,
		MediaType:       entry.MediaType,
		Text:            entry.Text,
		CreatedAt:       entry.CreatedAt,
	}
	return &out, nil
}

func (s *forwardAPIService) Stats(ctx context.Context, ownerUserID string) (*api.AutoForwardStats, error) {
	stats := &api.AutoForwardStats{}
	if s.store == nil || s.store.DB == nil {
		return stats, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.AutoForward{})
	if ownerUserID = strings.TrimSpace(ownerUserID); ownerUserID != "" {
		ownerID := parseUUIDPtr(ownerUserID)
		if ownerID == nil {
			return stats, nil
		}
		db = db.Where("owner_user_id = ?", *ownerID)
	}
	if err := db.Count(&stats.Total).Error; err != nil {
		if isMissingTableError(err) {
			return stats, nil
		}
		return nil, err
	}
	scoped := func(column string, value any) *gorm.DB {
		inner := s.store.DB.WithContext(ctx).Model(&model.AutoForward{})
		if ownerUserID != "" {
			inner = inner.Where("owner_user_id = ?", parseUUIDPtr(ownerUserID))
		}
		return inner.Where(column+" = ?", value)
	}
	if err := scoped("enabled", true).Count(&stats.Enabled).Error; err != nil && !isMissingTableError(err) {
		return nil, err
	}
	if err := scoped("realtime", true).Count(&stats.Realtime).Error; err != nil && !isMissingTableError(err) {
		return nil, err
	}
	if err := scoped("history_state", model.ForwardStateRunning).Count(&stats.Running).Error; err != nil && !isMissingTableError(err) {
		return nil, err
	}

	var totals struct {
		Forwarded int64
		Failed    int64
		Skipped   int64
	}
	var records []model.AutoForward
	if err := db.Select("total_forwarded", "total_failed", "total_skipped").Find(&records).Error; err != nil {
		if isMissingTableError(err) {
			return stats, nil
		}
		return nil, err
	}
	for _, record := range records {
		totals.Forwarded += record.TotalForwarded
		totals.Failed += record.TotalFailed
		totals.Skipped += record.TotalSkipped
	}
	stats.Forwarded = totals.Forwarded
	stats.Failed = totals.Failed
	stats.Skipped = totals.Skipped
	return stats, nil
}

// ---------------------------------------------------------------- helpers

func (s *forwardAPIService) findForward(ctx context.Context, id string, ownerUserID string) (*model.AutoForward, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, errors.New("forward storage is unavailable")
	}
	parsed, err := parseForwardID(id)
	if err != nil {
		return nil, err
	}
	var record model.AutoForward
	err = s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("forward rule not found")
	}
	if err != nil {
		return nil, err
	}
	if ownerUserID = strings.TrimSpace(ownerUserID); ownerUserID != "" {
		ownerID := parseUUIDPtr(ownerUserID)
		if ownerID == nil || record.OwnerUserID == nil || *record.OwnerUserID != *ownerID {
			return nil, api.ErrForbidden
		}
	}
	return &record, nil
}

func applyForwardRequest(record *model.AutoForward, req api.AutoForwardCreateRequest) {
	if req.Name != "" {
		record.Name = strings.TrimSpace(req.Name)
	}
	// 前端在更新时会回传完整对象，因此 enabled 以请求为准。
	record.Enabled = req.Enabled
	if req.SourceChatID != 0 {
		record.SourceChatID = req.SourceChatID
	}
	if req.SourceTitle != "" {
		record.SourceTitle = strings.TrimSpace(req.SourceTitle)
	}
	if req.TargetChatID != 0 {
		record.TargetChatID = req.TargetChatID
	}
	if req.TargetTitle != "" {
		record.TargetTitle = strings.TrimSpace(req.TargetTitle)
	}
	if req.Mode != "" {
		record.Mode = normalizeForwardMode(req.Mode)
	}
	// 前端更新时会回传完整对象，parse_mode 空值即“纯文本”
	record.ParseMode = normalizeForwardParseMode(req.ParseMode)
	if req.Realtime != nil {
		record.Realtime = *req.Realtime
	}
	if req.HistoryFromID != nil {
		record.HistoryFromID = *req.HistoryFromID
		if record.HistoryCursor == 0 {
			record.HistoryCursor = *req.HistoryFromID
		}
	}
	if req.HistoryToID != nil {
		record.HistoryToID = *req.HistoryToID
	}
	if req.HistoryAutoStopMisses != nil {
		record.HistoryAutoStopMisses = clampForwardInt(*req.HistoryAutoStopMisses, 1, 2000, 25)
	}
	if req.DelayMS != nil {
		record.DelayMS = clampForwardInt(*req.DelayMS, 0, 60000, 800)
	}
	if req.BatchSize != nil {
		record.BatchSize = clampForwardInt(*req.BatchSize, 1, 200, 20)
	}
	if req.AlbumBatch != nil {
		record.AlbumBatch = *req.AlbumBatch
	}
	if req.SkipKeywords != nil {
		record.SkipKeywords = strings.TrimSpace(*req.SkipKeywords)
	}
	if req.SkipForwarded != nil {
		record.SkipForwarded = *req.SkipForwarded
	}
	if req.SkipNoMedia != nil {
		record.SkipNoMedia = *req.SkipNoMedia
	}
	if req.SkipNoText != nil {
		record.SkipNoText = *req.SkipNoText
	}
	if req.SkipDuplicates != nil {
		record.SkipDuplicates = *req.SkipDuplicates
	}
	if req.MinFileSizeMB != nil {
		record.MinFileSizeMB = clampForwardInt(*req.MinFileSizeMB, 0, 2000, 0)
	}
	if req.MaxFileSizeMB != nil {
		record.MaxFileSizeMB = clampForwardInt(*req.MaxFileSizeMB, 0, 2000, 0)
	}
	if req.MinTextLength != nil {
		record.MinTextLength = clampForwardInt(*req.MinTextLength, 0, 10000, 0)
	}
	if req.CaptionTemplate != nil {
		record.CaptionTemplate = *req.CaptionTemplate
	}
	if req.CaptionPrefix != nil {
		record.CaptionPrefix = *req.CaptionPrefix
	}
	if req.CaptionSuffix != nil {
		record.CaptionSuffix = *req.CaptionSuffix
	}
	if req.RemoveLinks != nil {
		record.RemoveLinks = *req.RemoveLinks
	}
	if req.RemoveMentions != nil {
		record.RemoveMentions = *req.RemoveMentions
	}
	if req.RemoveHashtags != nil {
		record.RemoveHashtags = *req.RemoveHashtags
	}
	if req.DropSourceLine != nil {
		record.DropSourceLine = *req.DropSourceLine
	}
	if req.ReplacementsJSON != nil {
		record.ReplacementsJSON = strings.TrimSpace(*req.ReplacementsJSON)
	}
	if req.KeepOriginalText != nil {
		record.KeepOriginalText = *req.KeepOriginalText
	}
	if req.BuyEnabled != nil {
		record.BuyEnabled = *req.BuyEnabled
	}
	if req.BuyText != nil {
		record.BuyText = strings.TrimSpace(*req.BuyText)
		if record.BuyText == "" {
			record.BuyText = "🛒 立即购买"
		}
	}
	if req.BuyURLTemplate != nil {
		record.BuyURLTemplate = strings.TrimSpace(*req.BuyURLTemplate)
	}
	if req.ProductIDRegex != nil {
		record.ProductIDRegex = strings.TrimSpace(*req.ProductIDRegex)
	}
	if req.BuyButtonOnMedia != nil {
		record.BuyButtonOnMedia = *req.BuyButtonOnMedia
	}
	if req.BuyButtonOnText != nil {
		record.BuyButtonOnText = *req.BuyButtonOnText
	}
	if req.ExtraButtonsJSON != nil {
		record.ExtraButtonsJSON = strings.TrimSpace(*req.ExtraButtonsJSON)
	}
}

func normalizeForwardMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case model.ForwardModeForward:
		return model.ForwardModeForward
	default:
		return model.ForwardModeCopy
	}
}

func normalizeForwardParseMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "html":
		return "HTML"
	case "markdown", "markdownv2":
		return "MarkdownV2"
	default:
		return ""
	}
}

func clampForwardInt(value, minValue, maxValue, fallback int) int {
	if value < minValue {
		return fallback
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func parseForwardID(id string) (uint64, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return 0, errors.New("forward rule id is required")
	}
	parsed, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return 0, errors.New("forward rule id is invalid")
	}
	return parsed, nil
}

func toAPIAutoForward(record model.AutoForward) api.AutoForward {
	ownerUserID := ""
	if record.OwnerUserID != nil {
		ownerUserID = record.OwnerUserID.String()
	}
	return api.AutoForward{
		ID:                    record.ID,
		Name:                  record.Name,
		Enabled:               record.Enabled,
		SourceChatID:          record.SourceChatID,
		SourceTitle:           record.SourceTitle,
		TargetChatID:          record.TargetChatID,
		TargetTitle:           record.TargetTitle,
		Mode:                  record.Mode,
		ParseMode:             record.ParseMode,
		Realtime:              record.Realtime,
		HistoryFromID:         record.HistoryFromID,
		HistoryToID:           record.HistoryToID,
		HistoryCursor:         record.HistoryCursor,
		HistoryState:          record.HistoryState,
		HistorySent:           record.HistorySent,
		HistoryFailed:         record.HistoryFailed,
		HistorySkipped:        record.HistorySkipped,
		HistoryStartedAt:      record.HistoryStartedAt,
		HistoryFinishedAt:     record.HistoryFinishedAt,
		HistoryAutoStopMisses: record.HistoryAutoStopMisses,
		DelayMS:               record.DelayMS,
		BatchSize:             record.BatchSize,
		AlbumBatch:            record.AlbumBatch,
		SkipKeywords:          record.SkipKeywords,
		SkipForwarded:         record.SkipForwarded,
		SkipNoMedia:           record.SkipNoMedia,
		SkipNoText:            record.SkipNoText,
		SkipDuplicates:        record.SkipDuplicates,
		MinFileSizeMB:         record.MinFileSizeMB,
		MaxFileSizeMB:         record.MaxFileSizeMB,
		MinTextLength:         record.MinTextLength,
		CaptionTemplate:       record.CaptionTemplate,
		CaptionPrefix:         record.CaptionPrefix,
		CaptionSuffix:         record.CaptionSuffix,
		RemoveLinks:           record.RemoveLinks,
		RemoveMentions:        record.RemoveMentions,
		RemoveHashtags:        record.RemoveHashtags,
		DropSourceLine:        record.DropSourceLine,
		ReplacementsJSON:      record.ReplacementsJSON,
		KeepOriginalText:      record.KeepOriginalText,
		BuyEnabled:            record.BuyEnabled,
		BuyText:               record.BuyText,
		BuyURLTemplate:        record.BuyURLTemplate,
		ProductIDRegex:        record.ProductIDRegex,
		BuyButtonOnMedia:      record.BuyButtonOnMedia,
		BuyButtonOnText:       record.BuyButtonOnText,
		ExtraButtonsJSON:      record.ExtraButtonsJSON,
		TotalForwarded:        record.TotalForwarded,
		TotalFailed:           record.TotalFailed,
		TotalSkipped:          record.TotalSkipped,
		LastForwardAt:         record.LastForwardAt,
		LastError:             record.LastError,
		OwnerUserID:           ownerUserID,
		CreatedAt:             record.CreatedAt,
		UpdatedAt:             record.UpdatedAt,
	}
}
