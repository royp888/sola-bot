package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm/clause"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

const defaultWelcomeText = "欢迎 {name} 加入！"
const verifyPendingSetKey = "verify:pending"

type VerifyTimeout struct {
	ChatID     int64
	UserID     int64
	Challenge  bot.VerifyChallenge
	PendingKey string
}

type AdminService struct {
	store *store.Store
	redis *redis.Client
}

func NewAdminService(st *store.Store, redisClient *redis.Client) *AdminService {
	return &AdminService{store: st, redis: redisClient}
}

func (s *AdminService) GetConfig(ctx context.Context, chatID int64) (bot.ChatAdminConfig, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return defaultAdminConfig(chatID), nil
	}

	cfg := model.ChatAdminConfig{
		ChatID:             chatID,
		WelcomeText:        defaultWelcomeText,
		VerifyEnabled:      true,
		VerifyType:         "button",
		VerifyTimeout:      60,
		WarnLimit:          3,
		VerifyQuestion:     "",
		VerifyOptions:      "[]",
		VerifyCorrectIndex: -1,
		VerifyWhitelist:    "",
		VerifyDifficulty:   "medium",
	}
	db := s.store.DB.WithContext(ctx)
	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chat_id"}},
		DoNothing: true,
	}).Create(&cfg).Error
	if err != nil && !isDuplicateKeyError(err) {
		return bot.ChatAdminConfig{}, err
	}
	if err := db.First(&cfg, "chat_id = ?", chatID).Error; err != nil {
		return bot.ChatAdminConfig{}, err
	}
	normalizeAdminConfig(&cfg)
	return modelAdminConfigToBot(cfg), nil
}

func (s *AdminService) UpdateConfig(ctx context.Context, chatID int64, patch bot.ChatAdminConfigPatch) (bot.ChatAdminConfig, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		cfg := defaultAdminConfig(chatID)
		applyAdminConfigPatch(&cfg, patch)
		return cfg, nil
	}

	current, err := s.GetConfig(ctx, chatID)
	if err != nil {
		return bot.ChatAdminConfig{}, err
	}
	applyAdminConfigPatch(&current, patch)

	updates := map[string]any{
		"welcome_text":         current.WelcomeText,
		"verify_enabled":       current.VerifyEnabled,
		"verify_type":          current.VerifyType,
		"verify_timeout":       current.VerifyTimeout,
		"warn_limit":           current.WarnLimit,
		"verify_question":      current.VerifyQuestion,
		"verify_options":       current.VerifyOptions,
		"verify_correct_index": current.VerifyCorrectIndex,
		"verify_whitelist":     current.VerifyWhitelist,
		"verify_difficulty":    current.VerifyDifficulty,
		"rules_text":           current.RulesText,
		"updated_at":           time.Now(),
	}
	err = s.store.DB.WithContext(ctx).Model(&model.ChatAdminConfig{}).
		Where("chat_id = ?", chatID).
		Updates(updates).Error
	if err != nil {
		return bot.ChatAdminConfig{}, err
	}
	return current, nil
}

func (s *AdminService) ToggleVerify(ctx context.Context, chatID int64) (bot.ChatAdminConfig, error) {
	cfg, err := s.GetConfig(ctx, chatID)
	if err != nil {
		return bot.ChatAdminConfig{}, err
	}
	enabled := !cfg.VerifyEnabled
	return s.UpdateConfig(ctx, chatID, bot.ChatAdminConfigPatch{VerifyEnabled: &enabled})
}

func (s *AdminService) RecordBan(ctx context.Context, chatID int64, userID int64, operatorID int64, reason string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	return s.store.DB.WithContext(ctx).Create(&model.BanLog{
		UserID:   userID,
		ChatID:   chatID,
		Reason:   reason,
		BannedBy: operatorID,
		BannedAt: time.Now(),
	}).Error
}

func (s *AdminService) RecordUnban(ctx context.Context, chatID int64, userID int64, operatorID int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	now := time.Now()
	return s.store.DB.WithContext(ctx).Model(&model.BanLog{}).
		Where("chat_id = ? AND user_id = ? AND unbanned_at IS NULL", chatID, userID).
		Update("unbanned_at", now).Error
}

func (s *AdminService) RecordWarn(ctx context.Context, chatID int64, userID int64, operatorID int64, reason string) (int64, int, error) {
	cfg, err := s.GetConfig(ctx, chatID)
	if err != nil {
		return 0, 0, err
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return 1, cfg.WarnLimit, nil
	}
	if err := s.store.DB.WithContext(ctx).Create(&model.WarnRecord{
		UserID:    userID,
		ChatID:    chatID,
		Reason:    reason,
		WarnedBy:  operatorID,
		CreatedAt: time.Now(),
	}).Error; err != nil {
		return 0, 0, err
	}
	count, err := s.CountWarns(ctx, chatID, userID)
	return count, cfg.WarnLimit, err
}

func (s *AdminService) ClearWarns(ctx context.Context, chatID int64, userID int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	return s.store.DB.WithContext(ctx).Model(&model.WarnRecord{}).
		Where("chat_id = ? AND user_id = ? AND cleared = false", chatID, userID).
		Update("cleared", true).Error
}

func (s *AdminService) CountWarns(ctx context.Context, chatID int64, userID int64) (int64, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, nil
	}
	var count int64
	err := s.store.DB.WithContext(ctx).Model(&model.WarnRecord{}).
		Where("chat_id = ? AND user_id = ? AND cleared = false", chatID, userID).
		Count(&count).Error
	return count, err
}

func (s *AdminService) ListWarns(ctx context.Context, chatID int64, userID int64) ([]bot.WarnRecord, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []bot.WarnRecord{}, nil
	}
	var records []model.WarnRecord
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Order("created_at desc").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	out := make([]bot.WarnRecord, 0, len(records))
	for _, record := range records {
		out = append(out, modelWarnRecordToBot(record))
	}
	return out, nil
}

func (s *AdminService) ListBans(ctx context.Context, chatID int64, limit int) ([]bot.BanLog, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []bot.BanLog{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var records []model.BanLog
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Order("banned_at desc").
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	out := make([]bot.BanLog, 0, len(records))
	for _, record := range records {
		out = append(out, modelBanLogToBot(record))
	}
	return out, nil
}

func (s *AdminService) SetVerifyChallenge(ctx context.Context, chatID int64, userID int64, challenge bot.VerifyChallenge, ttl time.Duration) error {
	if s == nil || s.redis == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	if challenge.Attempts <= 0 {
		challenge.Attempts = 3
	}
	if challenge.ExpireAt.IsZero() {
		challenge.ExpireAt = time.Now().Add(ttl)
	}
	payload, err := json.Marshal(challenge)
	if err != nil {
		return err
	}
	pipe := s.redis.TxPipeline()
	pipe.Set(ctx, verifyKey(chatID, userID), payload, ttl)
	pipe.Set(ctx, verifyPendingKey(chatID, userID), payload, ttl+24*time.Hour)
	pipe.ZAdd(ctx, verifyPendingSetKey, redis.Z{
		Score:  float64(challenge.ExpireAt.Unix()),
		Member: verifyPendingMember(chatID, userID),
	})
	_, err = pipe.Exec(ctx)
	return err
}

func (s *AdminService) CheckVerifyChallenge(ctx context.Context, chatID int64, userID int64, answer string) (bot.VerifyCheckResult, error) {
	if s == nil || s.redis == nil {
		return bot.VerifyCheckResult{}, nil
	}
	key := verifyKey(chatID, userID)
	challenge, ok, err := s.GetVerifyChallenge(ctx, chatID, userID)
	if err != nil {
		return bot.VerifyCheckResult{}, err
	}
	if !ok {
		return bot.VerifyCheckResult{Expired: true}, nil
	}
	if challenge.Answer == answer {
		if err := s.ClearVerifyChallenge(ctx, chatID, userID); err != nil {
			return bot.VerifyCheckResult{}, err
		}
		return bot.VerifyCheckResult{OK: true, Challenge: challenge}, nil
	}

	challenge.Attempts--
	if challenge.Attempts <= 0 {
		if err := s.ClearVerifyChallenge(ctx, chatID, userID); err != nil {
			return bot.VerifyCheckResult{}, err
		}
		return bot.VerifyCheckResult{RemainingAttempts: 0, ShouldKick: true, Challenge: challenge}, nil
	}

	ttl, err := s.redis.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = time.Minute
	}
	payload, err := json.Marshal(challenge)
	if err != nil {
		return bot.VerifyCheckResult{}, err
	}
	if err := s.redis.Set(ctx, key, payload, ttl).Err(); err != nil {
		return bot.VerifyCheckResult{}, err
	}
	_ = s.redis.Set(ctx, verifyPendingKey(chatID, userID), payload, ttl+24*time.Hour).Err()
	return bot.VerifyCheckResult{RemainingAttempts: challenge.Attempts, Challenge: challenge}, nil
}

func (s *AdminService) GetVerifyChallenge(ctx context.Context, chatID int64, userID int64) (bot.VerifyChallenge, bool, error) {
	if s == nil || s.redis == nil {
		return bot.VerifyChallenge{}, false, nil
	}
	raw, err := s.redis.Get(ctx, verifyKey(chatID, userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return bot.VerifyChallenge{}, false, nil
		}
		return bot.VerifyChallenge{}, false, err
	}
	var challenge bot.VerifyChallenge
	if err := json.Unmarshal([]byte(raw), &challenge); err != nil {
		challenge = bot.VerifyChallenge{Answer: raw, Attempts: 3}
	}
	if challenge.Attempts <= 0 {
		challenge.Attempts = 3
	}
	return challenge, true, nil
}

func (s *AdminService) ClearVerifyChallenge(ctx context.Context, chatID int64, userID int64) error {
	if s == nil || s.redis == nil {
		return nil
	}
	pipe := s.redis.TxPipeline()
	pipe.Del(ctx, verifyKey(chatID, userID), verifyPendingKey(chatID, userID))
	pipe.ZRem(ctx, verifyPendingSetKey, verifyPendingMember(chatID, userID))
	_, err := pipe.Exec(ctx)
	return err
}

func (s *AdminService) DueVerifyTimeouts(ctx context.Context, now time.Time, limit int64) ([]VerifyTimeout, error) {
	if s == nil || s.redis == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	members, err := s.redis.ZRangeByScore(ctx, verifyPendingSetKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(now.Unix(), 10),
		Count: limit,
	}).Result()
	if err != nil {
		return nil, err
	}
	out := make([]VerifyTimeout, 0, len(members))
	for _, member := range members {
		chatID, userID, ok := parseVerifyPendingMember(member)
		if !ok {
			_ = s.redis.ZRem(ctx, verifyPendingSetKey, member).Err()
			continue
		}
		raw, err := s.redis.Get(ctx, verifyPendingKey(chatID, userID)).Result()
		if err != nil {
			if err == redis.Nil {
				_ = s.redis.ZRem(ctx, verifyPendingSetKey, member).Err()
				continue
			}
			return nil, err
		}
		var challenge bot.VerifyChallenge
		if err := json.Unmarshal([]byte(raw), &challenge); err != nil {
			_ = s.redis.ZRem(ctx, verifyPendingSetKey, member).Err()
			continue
		}
		if challenge.ExpireAt.After(now) {
			_ = s.redis.ZAdd(ctx, verifyPendingSetKey, redis.Z{Score: float64(challenge.ExpireAt.Unix()), Member: member}).Err()
			continue
		}
		out = append(out, VerifyTimeout{
			ChatID:     chatID,
			UserID:     userID,
			Challenge:  challenge,
			PendingKey: member,
		})
	}
	return out, nil
}

func (s *AdminService) RecordVerifyEvent(ctx context.Context, chatID int64, userID int64, eventType string, detail string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	now := time.Now()
	record := model.ViolationRecord{
		BaseModel: model.BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		UserID:        userID,
		ChatID:        chatID,
		ViolationType: eventType,
		ActionTaken:   detail,
		DetectedBy:    "verify",
	}
	err := s.store.DB.WithContext(ctx).Create(&record).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func (s *AdminService) GetVerifyStats(ctx context.Context, chatID int64) (bot.VerifyStats, error) {
	stats := bot.VerifyStats{}
	if s == nil || s.store == nil || s.store.DB == nil {
		return stats, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.ViolationRecord{}).Where("chat_id = ? AND detected_by = ?", chatID, "verify")

	// count passed
	var passed int64
	if err := db.Where("violation_type = ?", "verify_pass").Count(&passed).Error; err != nil && !isMissingTableError(err) {
		return stats, err
	}
	stats.TotalPassed = passed

	// count failed
	var failed int64
	if err := s.store.DB.WithContext(ctx).Model(&model.ViolationRecord{}).Where("chat_id = ? AND detected_by = ? AND violation_type = ?", chatID, "verify", "verify_fail").Count(&failed).Error; err != nil && !isMissingTableError(err) {
		return stats, err
	}
	stats.TotalFailed = failed

	// count timeout
	var timeout int64
	if err := s.store.DB.WithContext(ctx).Model(&model.ViolationRecord{}).Where("chat_id = ? AND detected_by = ? AND violation_type = ?", chatID, "verify", "verify_timeout").Count(&timeout).Error; err != nil && !isMissingTableError(err) {
		return stats, err
	}
	stats.TotalTimeout = timeout

	// count pending from redis
	if s.redis != nil {
		count, err := s.redis.ZCard(ctx, verifyPendingSetKey).Result()
		if err == nil {
			stats.PendingCount = count
		}
	}

	return stats, nil
}

func defaultAdminConfig(chatID int64) bot.ChatAdminConfig {
	return bot.ChatAdminConfig{
		ChatID:             chatID,
		WelcomeText:        defaultWelcomeText,
		VerifyEnabled:      true,
		VerifyType:         "button",
		VerifyTimeout:      60,
		WarnLimit:          3,
		VerifyQuestion:     "",
		VerifyOptions:      "[]",
		VerifyCorrectIndex: -1,
		VerifyWhitelist:    "",
		VerifyDifficulty:   "medium",
	}
}

func normalizeAdminConfig(cfg *model.ChatAdminConfig) {
	if cfg.WelcomeText == "" {
		cfg.WelcomeText = defaultWelcomeText
	}
	if cfg.VerifyTimeout <= 0 {
		cfg.VerifyTimeout = 60
	}
	if cfg.WarnLimit <= 0 {
		cfg.WarnLimit = 3
	}
	if strings.TrimSpace(cfg.VerifyType) == "" {
		cfg.VerifyType = "button"
	}
	if cfg.VerifyOptions == "" {
		cfg.VerifyOptions = "[]"
	}
	if cfg.VerifyCorrectIndex < 0 {
		cfg.VerifyCorrectIndex = -1
	}
	difficulty := strings.TrimSpace(cfg.VerifyDifficulty)
	if difficulty != "easy" && difficulty != "medium" && difficulty != "hard" {
		cfg.VerifyDifficulty = "medium"
	} else {
		cfg.VerifyDifficulty = difficulty
	}
}

func applyAdminConfigPatch(cfg *bot.ChatAdminConfig, patch bot.ChatAdminConfigPatch) {
	if patch.WelcomeText != nil {
		cfg.WelcomeText = *patch.WelcomeText
	}
	if patch.VerifyEnabled != nil {
		cfg.VerifyEnabled = *patch.VerifyEnabled
	}
	if patch.VerifyType != nil {
		cfg.VerifyType = strings.TrimSpace(*patch.VerifyType)
	}
	if patch.VerifyTimeout != nil && *patch.VerifyTimeout >= 0 {
		cfg.VerifyTimeout = *patch.VerifyTimeout
	}
	if patch.WarnLimit != nil && *patch.WarnLimit > 0 {
		cfg.WarnLimit = *patch.WarnLimit
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
	if patch.VerifyWhitelist != nil {
		cfg.VerifyWhitelist = strings.TrimSpace(*patch.VerifyWhitelist)
	}
	if patch.VerifyDifficulty != nil {
		cfg.VerifyDifficulty = strings.TrimSpace(*patch.VerifyDifficulty)
	}
	if patch.RulesText != nil {
		cfg.RulesText = *patch.RulesText
	}
}

func modelAdminConfigToBot(cfg model.ChatAdminConfig) bot.ChatAdminConfig {
	return bot.ChatAdminConfig{
		ChatID:             cfg.ChatID,
		WelcomeText:        cfg.WelcomeText,
		VerifyEnabled:      cfg.VerifyEnabled,
		VerifyType:         cfg.VerifyType,
		VerifyTimeout:      cfg.VerifyTimeout,
		WarnLimit:          cfg.WarnLimit,
		VerifyQuestion:     cfg.VerifyQuestion,
		VerifyOptions:      cfg.VerifyOptions,
		VerifyCorrectIndex: cfg.VerifyCorrectIndex,
		VerifyWhitelist:    cfg.VerifyWhitelist,
		VerifyDifficulty:   cfg.VerifyDifficulty,
		RulesText:          cfg.RulesText,
	}
}

func modelWarnRecordToBot(record model.WarnRecord) bot.WarnRecord {
	return bot.WarnRecord{
		ID:        record.ID,
		UserID:    record.UserID,
		ChatID:    record.ChatID,
		Reason:    record.Reason,
		WarnedBy:  record.WarnedBy,
		CreatedAt: record.CreatedAt,
		Cleared:   record.Cleared,
	}
}

func modelBanLogToBot(record model.BanLog) bot.BanLog {
	return bot.BanLog{
		ID:         record.ID,
		UserID:     record.UserID,
		ChatID:     record.ChatID,
		Reason:     record.Reason,
		BannedBy:   record.BannedBy,
		BannedAt:   record.BannedAt,
		UnbannedAt: record.UnbannedAt,
	}
}

func verifyKey(chatID int64, userID int64) string {
	return fmt.Sprintf("verify:%d:%d", chatID, userID)
}

func verifyPendingKey(chatID int64, userID int64) string {
	return fmt.Sprintf("verify:pending:%d:%d", chatID, userID)
}

func verifyPendingMember(chatID int64, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func parseVerifyPendingMember(member string) (int64, int64, bool) {
	parts := strings.Split(member, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	chatID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return chatID, userID, true
}

func (s *AdminService) RecordSeenUser(ctx context.Context, chatID int64, userID int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	err := s.store.DB.WithContext(ctx).Exec(
		"INSERT INTO seen_users(chat_id, user_id, seen_at) VALUES(?, ?, NOW()) ON CONFLICT(chat_id, user_id) DO NOTHING",
		chatID, userID,
	).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func (s *AdminService) ListSeenUsers(ctx context.Context, chatID int64) ([]int64, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, nil
	}
	var ids []int64
	err := s.store.DB.WithContext(ctx).Raw(
		"SELECT user_id FROM seen_users WHERE chat_id = ?", chatID,
	).Scan(&ids).Error
	if isMissingTableError(err) {
		return nil, nil
	}
	return ids, err
}

var _ bot.AdminService = (*AdminService)(nil)
