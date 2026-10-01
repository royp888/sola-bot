package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

// broadcastAPIService backs the mass-messaging screens in the admin UI.
// It owns the CRUD + "queue it" half; the actual fan-out happens in
// internal/worker/broadcast.go.
type broadcastAPIService struct {
	store *store.Store
}

func NewBroadcastAPIService(st *store.Store) api.BroadcastService {
	return &broadcastAPIService{store: st}
}

func (s *broadcastAPIService) Create(ctx context.Context, req api.BroadcastCreateRequest, ownerUserID string) (*api.Broadcast, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, errors.New("broadcast storage is unavailable")
	}

	targetType := normalizeBroadcastTargetType(req.TargetType)
	if targetType == "" {
		return nil, errors.New("target_type must be one of users, groups or both")
	}

	chatIDs := normalizeChatIDSlice(req.ChatIDs)
	if targetType == model.BroadcastTargetUsers && len(chatIDs) > 0 {
		return nil, errors.New("chat_ids cannot be used with target_type=users")
	}
	if err := s.ensureOwnedChatIDs(ctx, ownerUserID, chatIDs); err != nil {
		return nil, err
	}

	mediaURL, mediaName, mediaMime, mediaData, err := normalizeInlineMedia(
		strings.TrimSpace(req.MediaURL),
		strings.TrimSpace(req.MediaName),
		strings.TrimSpace(req.MediaMime),
		strings.TrimSpace(req.MediaDataBase64),
	)
	if err != nil {
		return nil, err
	}

	mediaType := normalizeScheduledPostMediaType(req.MediaType)
	content := strings.TrimSpace(req.Content)
	if content == "" && mediaURL == "" && len(mediaData) == 0 {
		return nil, errors.New("broadcast needs content or media")
	}

	keyboardJSON, err := marshalBroadcastKeyboard(req.InlineKeyboard, req.InlineKeyboardJSON)
	if err != nil {
		return nil, err
	}

	scheduledAt := req.ScheduledAt
	status := model.BroadcastStatusDraft
	if req.SendNow {
		now := time.Now()
		scheduledAt = &now
		status = model.BroadcastStatusQueued
	} else if scheduledAt != nil && !scheduledAt.After(time.Now()) {
		status = model.BroadcastStatusQueued
	}

	parseMode := strings.TrimSpace(req.ParseMode)
	if parseMode == "" {
		parseMode = "HTML"
	}

	record := model.Broadcast{
		Title:              strings.TrimSpace(req.Title),
		Content:            content,
		MediaURL:           mediaURL,
		MediaName:          mediaName,
		MediaMime:          mediaMime,
		MediaData:          mediaData,
		MediaType:          mediaType,
		ParseMode:          parseMode,
		InlineKeyboardJSON: keyboardJSON,
		TargetType:         targetType,
		ChatIDsJSON:        marshalChatIDs(chatIDs),
		Status:             status,
		ScheduledAt:        scheduledAt,
		OwnerUserID:        parseUUIDPtr(ownerUserID),
		CreatedByUserID:    parseUUIDPtr(ownerUserID),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.store.DB.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}

	out := toAPIBroadcast(record)
	// Persist the immediate target count so the列表 shows it before the worker runs.
	preview, previewErr := s.PreviewTargets(ctx, req, ownerUserID)
	if previewErr == nil && preview != nil {
		out.TotalTargets = int(preview.Total)
		if err := s.store.DB.WithContext(ctx).Model(&model.Broadcast{}).
			Where("id = ?", record.ID).
			Update("total_targets", preview.Total).Error; err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (s *broadcastAPIService) List(ctx context.Context, query api.CommonListQuery, ownerUserID string) ([]api.Broadcast, error) {
	if s.store == nil || s.store.DB == nil {
		return []api.Broadcast{}, nil
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	db := s.store.DB.WithContext(ctx).Model(&model.Broadcast{})
	if ownerID := parseUUIDPtr(ownerUserID); ownerID != nil {
		db = db.Where("owner_user_id = ?", *ownerID)
	}

	var records []model.Broadcast
	err := db.Order("created_at desc").Limit(limit).Offset(query.Offset).Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []api.Broadcast{}, nil
		}
		return nil, err
	}

	out := make([]api.Broadcast, 0, len(records))
	for _, record := range records {
		out = append(out, toAPIBroadcast(record))
	}
	return out, nil
}

func (s *broadcastAPIService) Get(ctx context.Context, id string, ownerUserID string) (*api.Broadcast, error) {
	record, err := s.findBroadcast(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	out := toAPIBroadcast(*record)
	return &out, nil
}

func (s *broadcastAPIService) Update(ctx context.Context, id string, req api.BroadcastUpdateRequest, ownerUserID string) (*api.Broadcast, error) {
	record, err := s.findBroadcast(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	if record.Status == model.BroadcastStatusSending || record.Status == model.BroadcastStatusCompleted {
		return nil, fmt.Errorf("broadcast %d can no longer be edited", record.ID)
	}

	updates := map[string]any{"updated_at": time.Now()}

	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Content != nil {
		updates["content"] = strings.TrimSpace(*req.Content)
	}
	if req.ParseMode != nil {
		mode := strings.TrimSpace(*req.ParseMode)
		if mode == "" {
			mode = "HTML"
		}
		updates["parse_mode"] = mode
	}
	if req.MediaType != nil {
		updates["media_type"] = normalizeScheduledPostMediaType(*req.MediaType)
	}
	if req.MediaURL != nil || req.MediaDataBase64 != nil || req.MediaName != nil || req.MediaMime != nil {
		mediaURL, mediaName, mediaMime, mediaData, err := normalizeInlineMedia(
			derefString(req.MediaURL),
			derefString(req.MediaName),
			derefString(req.MediaMime),
			derefString(req.MediaDataBase64),
		)
		if err != nil {
			return nil, err
		}
		updates["media_url"] = mediaURL
		updates["media_name"] = mediaName
		updates["media_mime"] = mediaMime
		updates["media_data"] = mediaData
	}
	if req.InlineKeyboard != nil || req.InlineKeyboardJSON != nil {
		keyboardJSON, err := marshalBroadcastKeyboard(
			derefKeyboard(req.InlineKeyboard),
			derefString(req.InlineKeyboardJSON),
		)
		if err != nil {
			return nil, err
		}
		updates["inline_keyboard_json"] = keyboardJSON
	}
	if req.TargetType != nil {
		targetType := normalizeBroadcastTargetType(*req.TargetType)
		if targetType == "" {
			return nil, errors.New("target_type must be one of users, groups or both")
		}
		updates["target_type"] = targetType
	}
	if req.ChatIDs != nil {
		chatIDs := normalizeChatIDSlice(*req.ChatIDs)
		if err := s.ensureOwnedChatIDs(ctx, ownerUserID, chatIDs); err != nil {
			return nil, err
		}
		updates["chat_ids_json"] = marshalChatIDs(chatIDs)
	}
	if req.ScheduledAt != nil {
		updates["scheduled_at"] = req.ScheduledAt
	}

	if err := s.store.DB.WithContext(ctx).Model(&model.Broadcast{}).
		Where("id = ?", record.ID).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.Get(ctx, strconv.FormatUint(record.ID, 10), ownerUserID)
}

func (s *broadcastAPIService) Delete(ctx context.Context, id string, ownerUserID string) error {
	record, err := s.findBroadcast(ctx, id, ownerUserID)
	if err != nil {
		return err
	}
	if record.Status == model.BroadcastStatusSending {
		return errors.New("cannot delete a broadcast that is currently sending")
	}
	return s.store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("broadcast_id = ?", record.ID).Delete(&model.BroadcastDelivery{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Broadcast{}, record.ID).Error
	})
}

// Send queues a draft for immediate delivery.
func (s *broadcastAPIService) Send(ctx context.Context, id string, ownerUserID string) (*api.Broadcast, error) {
	record, err := s.findBroadcast(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	switch record.Status {
	case model.BroadcastStatusSending:
		return nil, errors.New("broadcast is already sending")
	case model.BroadcastStatusCompleted:
		return nil, errors.New("broadcast has already been delivered")
	}

	preview, err := s.previewForRecord(ctx, *record, ownerUserID)
	if err != nil {
		return nil, err
	}
	if preview.Total == 0 {
		return nil, errors.New("no recipients matched this broadcast")
	}

	now := time.Now()
	updates := map[string]any{
		"status":        model.BroadcastStatusQueued,
		"scheduled_at":  now,
		"total_targets": int(preview.Total),
		"sent_count":    0,
		"failed_count":  0,
		"last_error":    "",
		"started_at":    nil,
		"finished_at":   nil,
		"updated_at":    now,
	}
	if err := s.store.DB.WithContext(ctx).Model(&model.Broadcast{}).
		Where("id = ?", record.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	// Clear any previous attempt so the detail view shows a fresh run.
	if err := s.store.DB.WithContext(ctx).
		Where("broadcast_id = ?", record.ID).
		Delete(&model.BroadcastDelivery{}).Error; err != nil {
		return nil, err
	}

	return s.Get(ctx, strconv.FormatUint(record.ID, 10), ownerUserID)
}

// Cancel stops a queued broadcast before the worker picks it up.
func (s *broadcastAPIService) Cancel(ctx context.Context, id string, ownerUserID string) (*api.Broadcast, error) {
	record, err := s.findBroadcast(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}
	if record.Status == model.BroadcastStatusSending {
		return nil, errors.New("broadcast is already sending and cannot be canceled")
	}
	if err := s.store.DB.WithContext(ctx).Model(&model.Broadcast{}).
		Where("id = ?", record.ID).
		Updates(map[string]any{
			"status":     model.BroadcastStatusCanceled,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, strconv.FormatUint(record.ID, 10), ownerUserID)
}

func (s *broadcastAPIService) Deliveries(ctx context.Context, id string, query api.CommonListQuery, ownerUserID string) ([]api.BroadcastDelivery, error) {
	record, err := s.findBroadcast(ctx, id, ownerUserID)
	if err != nil {
		return nil, err
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var rows []model.BroadcastDelivery
	err = s.store.DB.WithContext(ctx).
		Where("broadcast_id = ?", record.ID).
		Order("id asc").
		Limit(limit).
		Offset(query.Offset).
		Find(&rows).Error
	if err != nil {
		if isMissingTableError(err) {
			return []api.BroadcastDelivery{}, nil
		}
		return nil, err
	}

	out := make([]api.BroadcastDelivery, 0, len(rows))
	for _, row := range rows {
		out = append(out, api.BroadcastDelivery{
			ID:         row.ID,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			TargetName: row.TargetName,
			Status:     row.Status,
			MessageID:  row.MessageID,
			Error:      row.Error,
			SentAt:     row.SentAt,
			CreatedAt:  row.CreatedAt,
		})
	}
	return out, nil
}

func (s *broadcastAPIService) PreviewTargets(ctx context.Context, req api.BroadcastCreateRequest, ownerUserID string) (*api.BroadcastTargetPreview, error) {
	draft := model.Broadcast{
		TargetType:  normalizeBroadcastTargetType(req.TargetType),
		ChatIDsJSON: marshalChatIDs(normalizeChatIDSlice(req.ChatIDs)),
	}
	if draft.TargetType == "" {
		return nil, errors.New("target_type must be one of users, groups or both")
	}
	return s.previewForRecord(ctx, draft, ownerUserID)
}

func (s *broadcastAPIService) Stats(ctx context.Context, ownerUserID string) (*api.BroadcastStats, error) {
	if s.store == nil || s.store.DB == nil {
		return &api.BroadcastStats{}, nil
	}

	db := s.store.DB.WithContext(ctx).Model(&model.Broadcast{})
	if ownerID := parseUUIDPtr(ownerUserID); ownerID != nil {
		db = db.Where("owner_user_id = ?", *ownerID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		if isMissingTableError(err) {
			return &api.BroadcastStats{}, nil
		}
		return nil, err
	}

	countByStatus := func(status string) int64 {
		var n int64
		q := s.store.DB.WithContext(ctx).Model(&model.Broadcast{}).Where("status = ?", status)
		if ownerID := parseUUIDPtr(ownerUserID); ownerID != nil {
			q = q.Where("owner_user_id = ?", *ownerID)
		}
		if err := q.Count(&n).Error; err != nil {
			return 0
		}
		return n
	}

	return &api.BroadcastStats{
		Total:     total,
		Draft:     countByStatus(model.BroadcastStatusDraft),
		Queued:    countByStatus(model.BroadcastStatusQueued),
		Sending:   countByStatus(model.BroadcastStatusSending),
		Completed: countByStatus(model.BroadcastStatusCompleted),
		Failed:    countByStatus(model.BroadcastStatusFailed),
	}, nil
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

func (s *broadcastAPIService) findBroadcast(ctx context.Context, id string, ownerUserID string) (*model.Broadcast, error) {
	if s.store == nil || s.store.DB == nil {
		return nil, errors.New("broadcast storage is unavailable")
	}

	broadcastID, err := strconv.ParseUint(strings.TrimSpace(id), 10, 64)
	if err != nil {
		return nil, errors.New("invalid broadcast id")
	}

	var record model.Broadcast
	db := s.store.DB.WithContext(ctx).Where("id = ?", broadcastID)
	if ownerID := parseUUIDPtr(ownerUserID); ownerID != nil {
		db = db.Where("owner_user_id = ?", *ownerID)
	}
	if err := db.First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("broadcast %d not found", broadcastID)
		}
		return nil, err
	}
	return &record, nil
}

func (s *broadcastAPIService) previewForRecord(ctx context.Context, record model.Broadcast, ownerUserID string) (*api.BroadcastTargetPreview, error) {
	if s.store == nil || s.store.DB == nil {
		return &api.BroadcastTargetPreview{}, nil
	}

	out := &api.BroadcastTargetPreview{}

	if record.TargetType == model.BroadcastTargetUsers || record.TargetType == model.BroadcastTargetBoth {
		var n int64
		err := s.store.DB.WithContext(ctx).Model(&model.User{}).
			Where("telegram_user_id IS NOT NULL").
			Where("is_active = ?", true).
			Where("status = ?", "active").
			Count(&n).Error
		if err != nil {
			return nil, err
		}
		out.Users = n
	}

	if record.TargetType == model.BroadcastTargetGroups || record.TargetType == model.BroadcastTargetBoth {
		chatIDs, err := s.resolveChatScope(ctx, record.ChatIDsJSON, ownerUserID)
		if err != nil {
			return nil, err
		}
		if len(chatIDs) == 0 {
			// A broadcast must not silently reach nobody.
			out.Chats = 0
		} else {
			var n int64
			err = s.store.DB.WithContext(ctx).Model(&model.TelegramChat{}).
				Where("status = ?", "active").
				Where("telegram_chat_id IN ?", chatIDs).
				Count(&n).Error
			if err != nil {
				return nil, err
			}
			out.Chats = n
		}
	}

	out.Total = out.Users + out.Chats
	return out, nil
}

// resolveChatScope narrows the requested chat ids to the ones the caller owns.
func (s *broadcastAPIService) resolveChatScope(ctx context.Context, chatIDsJSON string, ownerUserID string) ([]int64, error) {
	requested := parseChatIDsJSON(chatIDsJSON)

	if strings.TrimSpace(ownerUserID) == "" {
		if len(requested) > 0 {
			return requested, nil
		}
		var all []int64
		err := s.store.DB.WithContext(ctx).Model(&model.TelegramChat{}).
			Where("status = ?", "active").
			Pluck("telegram_chat_id", &all).Error
		if err != nil {
			return nil, err
		}
		return all, nil
	}

	owned, err := ownedTelegramChatIDs(ctx, s.store, ownerUserID)
	if err != nil {
		return nil, err
	}
	if len(requested) == 0 {
		return owned, nil
	}

	allowed := make([]int64, 0, len(requested))
	for _, id := range requested {
		if containsTelegramChatID(owned, id) {
			allowed = append(allowed, id)
		}
	}
	return allowed, nil
}

func (s *broadcastAPIService) ensureOwnedChatIDs(ctx context.Context, ownerUserID string, chatIDs []int64) error {
	if len(chatIDs) == 0 {
		return nil
	}
	scope, err := s.resolveChatScope(ctx, marshalChatIDs(chatIDs), ownerUserID)
	if err != nil {
		return err
	}
	if len(scope) != len(chatIDs) {
		return api.ErrForbidden
	}
	return nil
}

func normalizeBroadcastTargetType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case model.BroadcastTargetUsers:
		return model.BroadcastTargetUsers
	case model.BroadcastTargetGroups, "group", "chats", "chat":
		return model.BroadcastTargetGroups
	case model.BroadcastTargetBoth, "all":
		return model.BroadcastTargetBoth
	default:
		return ""
	}
}

func normalizeChatIDSlice(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func marshalChatIDs(ids []int64) string {
	if len(ids) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func parseChatIDsJSON(raw string) []int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return ids
}

// marshalBroadcastKeyboard prefers the structured rows and falls back to the
// raw JSON string used by the older posts API.
func marshalBroadcastKeyboard(rows []api.InlineKeyboardRow, rawJSON string) (string, error) {
	if len(rows) == 0 && strings.TrimSpace(rawJSON) == "" {
		return "[]", nil
	}

	if len(rows) == 0 {
		trimmed := strings.TrimSpace(rawJSON)
		if trimmed == "" || trimmed == "[]" || trimmed == "null" {
			return "[]", nil
		}
		var parsed []api.InlineKeyboardRow
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return "", fmt.Errorf("inline_keyboard_json is not valid JSON: %w", err)
		}
		rows = parsed
	}

	type buttonPayload struct {
		Text         string `json:"text"`
		URL          string `json:"url,omitempty"`
		CallbackData string `json:"callback_data,omitempty"`
	}
	type rowPayload struct {
		Buttons []buttonPayload `json:"buttons"`
	}

	payload := make([]rowPayload, 0, len(rows))
	for _, row := range rows {
		buttons := make([]buttonPayload, 0, len(row.Buttons))
		for _, button := range row.Buttons {
			if strings.TrimSpace(button.Text) == "" {
				continue
			}
			buttons = append(buttons, buttonPayload{
				Text:         strings.TrimSpace(button.Text),
				URL:          strings.TrimSpace(button.URL),
				CallbackData: strings.TrimSpace(button.CallbackData),
			})
		}
		if len(buttons) == 0 {
			continue
		}
		payload = append(payload, rowPayload{Buttons: buttons})
	}

	if len(payload) == 0 {
		return "[]", nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func toAPIBroadcast(record model.Broadcast) api.Broadcast {
	chatIDs := parseChatIDsJSON(record.ChatIDsJSON)
	if chatIDs == nil {
		chatIDs = []int64{}
	}
	return api.Broadcast{
		ID:                 record.ID,
		Title:              record.Title,
		Content:            record.Content,
		MediaURL:           record.MediaURL,
		MediaName:          record.MediaName,
		MediaType:          record.MediaType,
		HasMedia:           len(record.MediaData) > 0 || strings.TrimSpace(record.MediaURL) != "",
		ParseMode:          record.ParseMode,
		InlineKeyboardJSON: record.InlineKeyboardJSON,
		TargetType:         record.TargetType,
		ChatIDs:            chatIDs,
		Status:             record.Status,
		ScheduledAt:        record.ScheduledAt,
		StartedAt:          record.StartedAt,
		FinishedAt:         record.FinishedAt,
		TotalTargets:       record.TotalTargets,
		SentCount:          record.SentCount,
		FailedCount:        record.FailedCount,
		LastError:          record.LastError,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}
}

func parseUUIDPtr(value string) *model.UUID {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil
	}
	return &parsed
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefKeyboard(value *[]api.InlineKeyboardRow) []api.InlineKeyboardRow {
	if value == nil {
		return nil
	}
	return *value
}
