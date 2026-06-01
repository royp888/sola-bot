package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
)

const pointConfigCacheTTL = 5 * time.Minute

func defaultPointConfig(chatID int64) bot.ChatPointConfig {
	return bot.ChatPointConfig{
		ChatID:          chatID,
		PointText:       1,
		PointPhoto:      3,
		PointSticker:    2,
		PointVideo:      3,
		PointFile:       2,
		PointVoice:      3,
		CooldownSeconds: 60,
		Enabled:         true,
	}
}

func (s *PointsService) GetConfig(ctx context.Context, chatID int64) (bot.ChatPointConfig, error) {
	cfg := defaultPointConfig(chatID)
	if chatID == 0 {
		return cfg, nil
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return cfg, nil
	}

	if cached, ok := s.getCachedPointConfig(ctx, chatID); ok {
		return cached, nil
	}

	record := modelPointConfig(cfg)
	db := s.store.DB.WithContext(ctx)
	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chat_id"}},
		DoNothing: true,
	}).Create(&record).Error
	if err != nil && !isDuplicateKeyError(err) {
		return bot.ChatPointConfig{}, err
	}
	if err := db.First(&record, "chat_id = ?", chatID).Error; err != nil {
		return bot.ChatPointConfig{}, err
	}

	cfg = botPointConfig(record)
	s.cachePointConfig(ctx, cfg)
	return cfg, nil
}

func (s *PointsService) UpdateConfig(ctx context.Context, chatID int64, patch bot.ChatPointConfigPatch) (bot.ChatPointConfig, error) {
	current, err := s.GetConfig(ctx, chatID)
	if err != nil {
		return bot.ChatPointConfig{}, err
	}
	applyPointConfigPatch(&current, patch)

	if s == nil || s.store == nil || s.store.DB == nil {
		return current, nil
	}

	updates := map[string]any{
		"point_text":       current.PointText,
		"point_photo":      current.PointPhoto,
		"point_sticker":    current.PointSticker,
		"point_video":      current.PointVideo,
		"point_file":       current.PointFile,
		"point_voice":      current.PointVoice,
		"cooldown_seconds": current.CooldownSeconds,
		"enabled":          current.Enabled,
	}
	if err := s.store.DB.WithContext(ctx).
		Model(&model.ChatPointConfig{}).
		Where("chat_id = ?", chatID).
		Updates(updates).Error; err != nil {
		return bot.ChatPointConfig{}, err
	}

	s.invalidatePointConfig(ctx, chatID)
	s.cachePointConfig(ctx, current)
	return current, nil
}

func (s *PointsService) ToggleConfig(ctx context.Context, chatID int64) (bot.ChatPointConfig, error) {
	current, err := s.GetConfig(ctx, chatID)
	if err != nil {
		return bot.ChatPointConfig{}, err
	}
	enabled := !current.Enabled
	return s.UpdateConfig(ctx, chatID, bot.ChatPointConfigPatch{Enabled: &enabled})
}

func (s *PointsService) GetSummary(ctx context.Context, chatID int64, userID int64) (string, error) {
	point, err := s.GetUserPoint(ctx, chatID, userID)
	if err != nil {
		return "", err
	}
	level, err := s.GetLevelByPoints(ctx, chatID, userID, point.TotalPoints)
	if err != nil {
		return fmt.Sprintf("当前积分：%d", point.TotalPoints), nil
	}
	return fmt.Sprintf("当前积分：%d\n%s", point.TotalPoints, formatUserLevel(level)), nil
}

func (s *PointsService) GetLevel(ctx context.Context, chatID int64, userID int64) (UserLevel, error) {
	return s.levelService().GetUserLevel(ctx, chatID, userID)
}

func (s *PointsService) GetLevelByPoints(ctx context.Context, chatID int64, userID int64, points int64) (UserLevel, error) {
	return s.levelService().GetUserLevelByPoints(ctx, chatID, userID, points)
}

func (s *PointsService) CheckLevelUpgrade(ctx context.Context, chatID int64, userID int64, previousPoints int64, currentPoints int64) (LevelUpgradeResult, error) {
	return s.levelService().CheckUpgrade(ctx, chatID, userID, previousPoints, currentPoints)
}

func (s *PointsService) GetRank(ctx context.Context, chatID int64, period string, limit int) (string, error) {
	entries, err := s.GetRankEntries(ctx, chatID, period, limit)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return rankPeriodLabel(period) + "暂无积分排行。", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s积分榜\n", rankPeriodLabel(period))
	for _, entry := range entries {
		fmt.Fprintf(&b, "%d. 用户 %d - %d\n", entry.Rank, entry.UserID, entry.Points)
	}
	return strings.TrimSpace(b.String()), nil
}

type PointRankEntry struct {
	Rank   int
	UserID int64
	Points int64
}

func (s *PointsService) GetRankEntries(ctx context.Context, chatID int64, period string, limit int) ([]PointRankEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if entries, ok := s.getRankEntriesFromRedis(ctx, chatID, period, limit); ok {
		return entries, nil
	}
	return s.getRankEntriesFromDB(ctx, chatID, period, limit)
}

func (s *PointsService) Adjust(ctx context.Context, chatID int64, userID int64, delta int, reason string) error {
	_, err := s.AdjustUserPoints(ctx, chatID, userID, delta, reason)
	return err
}

func (s *PointsService) GetActivityStats(ctx context.Context, chatID int64, window string) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "活跃统计服务尚未接入。", nil
	}
	var count int64
	since := time.Now().Add(-24 * time.Hour)
	if window == "week" {
		since = time.Now().Add(-7 * 24 * time.Hour)
	}
	if err := s.store.DB.WithContext(ctx).Model(&model.PointLog{}).Where("chat_id = ? AND created_at >= ?", chatID, since).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s 活跃事件：%d", window, count), nil
}

func (s *PointsService) AwardMessage(ctx context.Context, req bot.PointAwardRequest) (bot.PointAwardResult, error) {
	if req.FromBot || req.IsCommand || req.ChatID == 0 || req.UserID == 0 {
		return bot.PointAwardResult{Reason: "ignored"}, nil
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return bot.PointAwardResult{Reason: "store unavailable"}, nil
	}

	cfg, err := s.GetConfig(ctx, req.ChatID)
	if err != nil {
		return bot.PointAwardResult{}, err
	}
	if !cfg.Enabled {
		return bot.PointAwardResult{Reason: "disabled"}, nil
	}

	if cooledDown, err := s.consumePointCooldown(ctx, req.ChatID, req.UserID, cfg.CooldownSeconds, req.CooldownScope); err != nil {
		return bot.PointAwardResult{}, err
	} else if !cooledDown {
		return bot.PointAwardResult{Reason: "cooldown"}, nil
	}

	delta := pointsForMessageType(cfg, req.MessageType)
	if delta <= 0 {
		return bot.PointAwardResult{Reason: "message type disabled"}, nil
	}

	reasonPrefix := strings.TrimSpace(req.ReasonPrefix)
	if reasonPrefix == "" {
		reasonPrefix = "message"
	}
	reason := reasonPrefix + ":" + req.MessageType
	if err := s.addUserPoints(ctx, req.ChatID, req.UserID, delta, reason); err != nil {
		return bot.PointAwardResult{}, err
	}
	s.incrementRankCaches(ctx, req.ChatID, req.UserID, delta, time.Now())

	return bot.PointAwardResult{Awarded: true, Points: delta, Reason: reason}, nil
}

func (s *PointsService) addUserPoints(ctx context.Context, chatID int64, userID int64, delta int, reason string) error {
	now := time.Now()
	return s.store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userPoint := model.UserPoint{
			UserID:      userID,
			ChatID:      chatID,
			TotalPoints: int64(delta),
			UpdatedAt:   now,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "chat_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"total_points": gorm.Expr("user_points.total_points + ?", delta),
				"updated_at":   now,
			}),
		}).Create(&userPoint).Error; err != nil {
			return err
		}
		return tx.Create(&model.PointLog{
			UserID:    userID,
			ChatID:    chatID,
			Delta:     delta,
			Reason:    truncateReason(reason),
			CreatedAt: now,
		}).Error
	})
}

func (s *PointsService) getCachedPointConfig(ctx context.Context, chatID int64) (bot.ChatPointConfig, bool) {
	if s == nil || s.store == nil || s.store.Redis == nil {
		return bot.ChatPointConfig{}, false
	}
	raw, err := s.store.Redis.Get(ctx, pointConfigCacheKey(chatID)).Result()
	if err != nil {
		return bot.ChatPointConfig{}, false
	}
	var cfg bot.ChatPointConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return bot.ChatPointConfig{}, false
	}
	return cfg, true
}

func (s *PointsService) cachePointConfig(ctx context.Context, cfg bot.ChatPointConfig) {
	if s == nil || s.store == nil || s.store.Redis == nil {
		return
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	_ = s.store.Redis.Set(ctx, pointConfigCacheKey(cfg.ChatID), raw, pointConfigCacheTTL).Err()
}

func (s *PointsService) invalidatePointConfig(ctx context.Context, chatID int64) {
	if s == nil || s.store == nil || s.store.Redis == nil {
		return
	}
	_ = s.store.Redis.Del(ctx, pointConfigCacheKey(chatID)).Err()
}

func (s *PointsService) consumePointCooldown(ctx context.Context, chatID int64, userID int64, cooldownSeconds int, scope string) (bool, error) {
	if cooldownSeconds <= 0 || s == nil || s.store == nil || s.store.Redis == nil {
		return true, nil
	}
	ok, err := s.store.Redis.SetNX(ctx, pointCooldownKey(chatID, userID, scope), "1", time.Duration(cooldownSeconds)*time.Second).Result()
	if err == redis.Nil || err != nil {
		return true, nil
	}
	return ok, nil
}

func pointCooldownKey(chatID int64, userID int64, scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" || scope == "message" {
		return fmt.Sprintf("cooldown:%d:%d", chatID, userID)
	}
	return fmt.Sprintf("cooldown:%s:%d:%d", scope, chatID, userID)
}

func pointConfigCacheKey(chatID int64) string {
	return fmt.Sprintf("config:%d", chatID)
}

func applyPointConfigPatch(cfg *bot.ChatPointConfig, patch bot.ChatPointConfigPatch) {
	if patch.PointText != nil {
		cfg.PointText = nonNegative(*patch.PointText)
	}
	if patch.PointPhoto != nil {
		cfg.PointPhoto = nonNegative(*patch.PointPhoto)
	}
	if patch.PointSticker != nil {
		cfg.PointSticker = nonNegative(*patch.PointSticker)
	}
	if patch.PointVideo != nil {
		cfg.PointVideo = nonNegative(*patch.PointVideo)
	}
	if patch.PointFile != nil {
		cfg.PointFile = nonNegative(*patch.PointFile)
	}
	if patch.PointVoice != nil {
		cfg.PointVoice = nonNegative(*patch.PointVoice)
	}
	if patch.CooldownSeconds != nil {
		cfg.CooldownSeconds = nonNegative(*patch.CooldownSeconds)
	}
	if patch.Enabled != nil {
		cfg.Enabled = *patch.Enabled
	}
}

func pointsForMessageType(cfg bot.ChatPointConfig, messageType string) int {
	switch messageType {
	case "text":
		return cfg.PointText
	case "photo":
		return cfg.PointPhoto
	case "sticker":
		return cfg.PointSticker
	case "video":
		return cfg.PointVideo
	case "file":
		return cfg.PointFile
	case "voice":
		return cfg.PointVoice
	default:
		return 0
	}
}

func modelPointConfig(cfg bot.ChatPointConfig) model.ChatPointConfig {
	return model.ChatPointConfig{
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

func botPointConfig(record model.ChatPointConfig) bot.ChatPointConfig {
	return bot.ChatPointConfig{
		ChatID:          record.ChatID,
		PointText:       record.PointText,
		PointPhoto:      record.PointPhoto,
		PointSticker:    record.PointSticker,
		PointVideo:      record.PointVideo,
		PointFile:       record.PointFile,
		PointVoice:      record.PointVoice,
		CooldownSeconds: record.CooldownSeconds,
		Enabled:         record.Enabled,
	}
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

type rankKeys struct {
	all   string
	day   string
	week  string
	month string
}

func (s *PointsService) incrementRankCaches(ctx context.Context, chatID int64, userID int64, delta int, now time.Time) {
	if s == nil || s.store == nil || s.store.Redis == nil || delta == 0 {
		return
	}
	member := strconv.FormatInt(userID, 10)
	keys := rankCacheKeys(chatID, now)
	pipe := s.store.Redis.Pipeline()
	pipe.ZIncrBy(ctx, keys.all, float64(delta), member)
	pipe.ZIncrBy(ctx, keys.day, float64(delta), member)
	pipe.Expire(ctx, keys.day, 25*time.Hour)
	pipe.ZIncrBy(ctx, keys.week, float64(delta), member)
	pipe.Expire(ctx, keys.week, 8*24*time.Hour)
	pipe.ZIncrBy(ctx, keys.month, float64(delta), member)
	pipe.Expire(ctx, keys.month, 32*24*time.Hour)
	_, _ = pipe.Exec(ctx)
}

func (s *PointsService) getRankEntriesFromRedis(ctx context.Context, chatID int64, period string, limit int) ([]PointRankEntry, bool) {
	if s == nil || s.store == nil || s.store.Redis == nil {
		return nil, false
	}
	key := rankCacheKey(chatID, period, time.Now())
	items, err := s.store.Redis.ZRevRangeWithScores(ctx, key, 0, int64(limit)-1).Result()
	if err != nil || len(items) == 0 {
		return nil, false
	}
	entries := make([]PointRankEntry, 0, len(items))
	for i, item := range items {
		userID, err := strconv.ParseInt(fmt.Sprint(item.Member), 10, 64)
		if err != nil {
			continue
		}
		entries = append(entries, PointRankEntry{Rank: i + 1, UserID: userID, Points: int64(item.Score)})
	}
	return entries, len(entries) > 0
}

func (s *PointsService) getRankEntriesFromDB(ctx context.Context, chatID int64, period string, limit int) ([]PointRankEntry, error) {
	if period == "all" {
		var rows []model.UserPoint
		if err := s.store.DB.WithContext(ctx).
			Where("chat_id = ?", chatID).
			Order("total_points desc").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]PointRankEntry, 0, len(rows))
		for i, row := range rows {
			out = append(out, PointRankEntry{Rank: i + 1, UserID: row.UserID, Points: row.TotalPoints})
		}
		return out, nil
	}

	start, end := rankPeriodRange(time.Now(), period)
	var rows []struct {
		UserID int64
		Total  int64
	}
	if err := s.store.DB.WithContext(ctx).
		Model(&model.PointLog{}).
		Select("user_id, COALESCE(SUM(delta), 0) AS total").
		Where("chat_id = ? AND created_at >= ? AND created_at < ?", chatID, start, end).
		Group("user_id").
		Order("total desc").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PointRankEntry, 0, len(rows))
	for i, row := range rows {
		out = append(out, PointRankEntry{Rank: i + 1, UserID: row.UserID, Points: row.Total})
	}
	return out, nil
}

func (s *PointsService) ListPointLogs(ctx context.Context, chatID int64, userID int64, limit int, offset int) ([]model.PointLog, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []model.PointLog{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var logs []model.PointLog
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

func (s *PointsService) GetUserPoint(ctx context.Context, chatID int64, userID int64) (model.UserPoint, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return model.UserPoint{ChatID: chatID, UserID: userID}, nil
	}
	var point model.UserPoint
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		First(&point).Error
	if err == gorm.ErrRecordNotFound {
		return model.UserPoint{ChatID: chatID, UserID: userID}, nil
	}
	return point, err
}

func (s *PointsService) AdjustUserPoints(ctx context.Context, chatID int64, userID int64, delta int, reason string) (model.UserPoint, error) {
	point, _, err := s.AdjustUserPointsWithLevel(ctx, chatID, userID, delta, reason)
	return point, err
}

func (s *PointsService) AdjustUserPointsWithLevel(ctx context.Context, chatID int64, userID int64, delta int, reason string) (model.UserPoint, LevelUpgradeResult, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		point := model.UserPoint{ChatID: chatID, UserID: userID, TotalPoints: int64(delta)}
		upgrade, err := NewLevelService(nil).CheckUpgrade(ctx, chatID, userID, 0, point.TotalPoints)
		return point, upgrade, err
	}
	if strings.TrimSpace(reason) == "" {
		reason = "manual"
	}
	previous, err := s.GetUserPoint(ctx, chatID, userID)
	if err != nil {
		return model.UserPoint{}, LevelUpgradeResult{}, err
	}
	if err := s.addUserPoints(ctx, chatID, userID, delta, reason); err != nil {
		return model.UserPoint{}, LevelUpgradeResult{}, err
	}
	s.incrementRankCaches(ctx, chatID, userID, delta, time.Now())
	current, err := s.GetUserPoint(ctx, chatID, userID)
	if err != nil {
		return model.UserPoint{}, LevelUpgradeResult{}, err
	}
	upgrade, err := s.CheckLevelUpgrade(ctx, chatID, userID, previous.TotalPoints, current.TotalPoints)
	if err != nil {
		return model.UserPoint{}, LevelUpgradeResult{}, err
	}
	return current, upgrade, nil
}

func rankCacheKeys(chatID int64, now time.Time) rankKeys {
	year, week := now.ISOWeek()
	return rankKeys{
		all:   fmt.Sprintf("rank:all:%d", chatID),
		day:   fmt.Sprintf("rank:day:%d:%s", chatID, now.Format("20060102")),
		week:  fmt.Sprintf("rank:week:%d:%04d-%02d", chatID, year, week),
		month: fmt.Sprintf("rank:month:%d:%s", chatID, now.Format("2006-01")),
	}
}

func rankCacheKey(chatID int64, period string, now time.Time) string {
	keys := rankCacheKeys(chatID, now)
	switch normalizeRankPeriod(period) {
	case "day":
		return keys.day
	case "week":
		return keys.week
	case "month":
		return keys.month
	default:
		return keys.all
	}
}

func normalizeRankPeriod(period string) string {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "day", "today":
		return "day"
	case "week":
		return "week"
	case "month":
		return "month"
	default:
		return "all"
	}
}

func rankPeriodRange(now time.Time, period string) (time.Time, time.Time) {
	loc := now.Location()
	switch normalizeRankPeriod(period) {
	case "day":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return start, start.Add(24 * time.Hour)
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDay := now.AddDate(0, 0, -(weekday - 1))
		start := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 0, 7)
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 1, 0)
	default:
		return time.Time{}, now
	}
}

func rankPeriodLabel(period string) string {
	switch normalizeRankPeriod(period) {
	case "day":
		return "今日"
	case "week":
		return "本周"
	case "month":
		return "本月"
	default:
		return "全部时间"
	}
}

func truncateReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > 64 {
		return reason[:64]
	}
	return reason
}

func (s *PointsService) levelService() *LevelService {
	if s == nil {
		return NewLevelService(nil)
	}
	return NewLevelService(s.store)
}

func formatUserLevel(level UserLevel) string {
	if level.NextLevel == nil {
		return fmt.Sprintf("当前等级：%s %s（最高等级）", level.Badge, level.Name)
	}
	return fmt.Sprintf("当前等级：%s %s（距 %s 还差 %d 积分）", level.Badge, level.Name, level.NextLevel.Name, level.PointsToNext)
}
