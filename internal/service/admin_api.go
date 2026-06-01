package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
	"gorm.io/gorm"
)

type adminAPIService struct {
	admin *AdminService
}

func (s *adminAPIService) GetConfig(ctx context.Context, chatID int64) (*api.ChatAdminConfig, error) {
	if s == nil || s.admin == nil {
		cfg := bot.ChatAdminConfig{ChatID: chatID, WelcomeText: defaultWelcomeText, VerifyEnabled: true, VerifyTimeout: 60, WarnLimit: 3}
		return adminConfigToAPI(cfg), nil
	}
	cfg, err := s.admin.GetConfig(ctx, chatID)
	if err != nil {
		return nil, err
	}
	return adminConfigToAPI(cfg), nil
}

func (s *adminAPIService) UpdateConfig(ctx context.Context, chatID int64, req api.ChatAdminConfigUpdateRequest) (*api.ChatAdminConfig, error) {
	if s == nil || s.admin == nil {
		cfg := bot.ChatAdminConfig{ChatID: chatID, WelcomeText: defaultWelcomeText, VerifyEnabled: true, VerifyTimeout: 60, WarnLimit: 3}
		applyAPIAdminPatch(&cfg, req)
		return adminConfigToAPI(cfg), nil
	}
	cfg, err := s.admin.UpdateConfig(ctx, chatID, bot.ChatAdminConfigPatch{
		WelcomeText:   req.WelcomeText,
		VerifyEnabled: req.VerifyEnabled,
		VerifyTimeout: req.VerifyTimeout,
		WarnLimit:     req.WarnLimit,
	})
	if err != nil {
		return nil, err
	}
	return adminConfigToAPI(cfg), nil
}

func (s *adminAPIService) ListBans(ctx context.Context, chatID int64, query api.CommonListQuery) ([]api.BanLog, error) {
	if s == nil || s.admin == nil {
		return []api.BanLog{}, nil
	}
	items, err := s.admin.ListBans(ctx, chatID, query.Limit)
	if err != nil {
		return nil, err
	}
	out := make([]api.BanLog, 0, len(items))
	for _, item := range items {
		out = append(out, adminBanToAPI(item))
	}
	return out, nil
}

func (s *adminAPIService) Ban(ctx context.Context, req api.AdminBanRequest) error {
	if s == nil || s.admin == nil {
		return nil
	}
	return s.admin.RecordBan(ctx, req.ChatID, req.UserID, req.BannedBy, req.Reason)
}

func (s *adminAPIService) Unban(ctx context.Context, chatID int64, userID int64) error {
	if s == nil || s.admin == nil {
		return nil
	}
	return s.admin.RecordUnban(ctx, chatID, userID, 0)
}

func (s *adminAPIService) ListWarns(ctx context.Context, chatID int64, userID int64) ([]api.WarnRecord, error) {
	if s == nil || s.admin == nil {
		return []api.WarnRecord{}, nil
	}
	items, err := s.admin.ListWarns(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]api.WarnRecord, 0, len(items))
	for _, item := range items {
		out = append(out, adminWarnToAPI(item))
	}
	return out, nil
}

func (s *adminAPIService) ExportUserRows(ctx context.Context, query api.ExportUserQuery) ([]api.ExportUserRow, error) {
	if s == nil || s.admin == nil || s.admin.store == nil || s.admin.store.DB == nil {
		return []api.ExportUserRow{}, nil
	}
	db := s.admin.store.DB.WithContext(ctx).Model(&model.UserPoint{})
	if query.ChatID != 0 {
		db = db.Where("chat_id = ?", query.ChatID)
	}
	var points []model.UserPoint
	if err := db.Order("total_points desc").Limit(10000).Find(&points).Error; err != nil {
		if isMissingTableError(err) {
			return []api.ExportUserRow{}, nil
		}
		return nil, err
	}

	rows := make([]api.ExportUserRow, 0, len(points))
	for _, point := range points {
		row := api.ExportUserRow{
			UserID:      point.UserID,
			Username:    strconv.FormatInt(point.UserID, 10),
			DisplayName: fmt.Sprintf("User %d", point.UserID),
			ChatID:      point.ChatID,
			TotalPoints: point.TotalPoints,
			Level:       levelNameForPoints(ctx, s.admin.store.DB, point.ChatID, point.TotalPoints),
			Status:      "active",
			JoinedAt:    point.UpdatedAt,
			LastSeenAt:  point.UpdatedAt,
		}
		var user model.User
		err := s.admin.store.DB.WithContext(ctx).Where("telegram_user_id = ?", point.UserID).First(&user).Error
		if err == nil {
			if user.Username != nil && strings.TrimSpace(*user.Username) != "" {
				row.Username = *user.Username
			}
			if strings.TrimSpace(user.DisplayName) != "" {
				row.DisplayName = user.DisplayName
			}
			if strings.TrimSpace(user.Status) != "" {
				row.Status = normalizeUserStatus(user.Status)
			}
			row.JoinedAt = user.CreatedAt
			if user.LastLoginAt != nil {
				row.LastSeenAt = *user.LastLoginAt
			} else if !user.UpdatedAt.IsZero() {
				row.LastSeenAt = user.UpdatedAt
			}
		}
		warns, err := s.admin.CountWarns(ctx, point.ChatID, point.UserID)
		if err == nil {
			row.WarnCount = int(warns)
		}
		if query.Status != "" && !strings.EqualFold(row.Status, strings.TrimSpace(query.Status)) {
			continue
		}
		if !matchesExportUserKeyword(row, query.Keyword) {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *adminAPIService) BatchUserAction(ctx context.Context, req api.BatchUserRequest) (*api.BatchUserResult, error) {
	result := &api.BatchUserResult{Failed: []string{}}
	if s == nil || s.admin == nil {
		return result, nil
	}
	if len(req.UserIDs) == 0 {
		return result, fmt.Errorf("未选择用户")
	}
	if len(req.UserIDs) > 200 {
		return result, fmt.Errorf("单次批量操作上限 200 人")
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "batch_" + action
	}
	switch action {
	case "ban":
		for _, userID := range req.UserIDs {
			if err := s.admin.RecordBan(ctx, req.ChatID, userID, 0, reason); err != nil {
				result.Failed = append(result.Failed, fmt.Sprintf("%d: %v", userID, err))
				continue
			}
			result.SuccessCount++
		}
	case "adjust_points":
		points := NewPointsService(s.admin.store)
		for _, userID := range req.UserIDs {
			if err := points.Adjust(ctx, req.ChatID, userID, req.Delta, reason); err != nil {
				result.Failed = append(result.Failed, fmt.Sprintf("%d: %v", userID, err))
				continue
			}
			result.SuccessCount++
		}
	default:
		return result, fmt.Errorf("unsupported action %s", req.Action)
	}
	return result, nil
}

func adminConfigToAPI(cfg bot.ChatAdminConfig) *api.ChatAdminConfig {
	return &api.ChatAdminConfig{
		ChatID:        cfg.ChatID,
		WelcomeText:   cfg.WelcomeText,
		VerifyEnabled: cfg.VerifyEnabled,
		VerifyTimeout: cfg.VerifyTimeout,
		WarnLimit:     cfg.WarnLimit,
	}
}

func applyAPIAdminPatch(cfg *bot.ChatAdminConfig, req api.ChatAdminConfigUpdateRequest) {
	if req.WelcomeText != nil {
		cfg.WelcomeText = *req.WelcomeText
	}
	if req.VerifyEnabled != nil {
		cfg.VerifyEnabled = *req.VerifyEnabled
	}
	if req.VerifyTimeout != nil {
		cfg.VerifyTimeout = *req.VerifyTimeout
	}
	if req.WarnLimit != nil {
		cfg.WarnLimit = *req.WarnLimit
	}
}

func adminBanToAPI(record bot.BanLog) api.BanLog {
	return api.BanLog{
		ID:         record.ID,
		UserID:     record.UserID,
		ChatID:     record.ChatID,
		Reason:     record.Reason,
		BannedBy:   record.BannedBy,
		BannedAt:   record.BannedAt,
		UnbannedAt: record.UnbannedAt,
	}
}

func adminWarnToAPI(record bot.WarnRecord) api.WarnRecord {
	return api.WarnRecord{
		ID:        record.ID,
		UserID:    record.UserID,
		ChatID:    record.ChatID,
		Reason:    record.Reason,
		WarnedBy:  record.WarnedBy,
		CreatedAt: record.CreatedAt,
		Cleared:   record.Cleared,
	}
}

var _ api.ChatAdminService = (*adminAPIService)(nil)

func matchesExportUserKeyword(row api.ExportUserRow, keyword string) bool {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return true
	}
	return strings.Contains(strings.ToLower(row.Username), keyword) ||
		strings.Contains(strings.ToLower(row.DisplayName), keyword) ||
		strings.Contains(strconv.FormatInt(row.UserID, 10), keyword)
}

func levelNameForPoints(ctx context.Context, db *gorm.DB, chatID int64, points int64) string {
	if db == nil {
		return ""
	}
	var level model.LevelConfig
	err := db.WithContext(ctx).
		Where("chat_id = ? AND min_points <= ?", chatID, points).
		Order("min_points desc").
		First(&level).Error
	if err != nil || strings.TrimSpace(level.Label) == "" {
		return ""
	}
	return level.Label
}
