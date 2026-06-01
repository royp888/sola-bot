package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

type LevelService struct {
	store *store.Store
}

func NewLevelService(st *store.Store) *LevelService {
	return &LevelService{store: st}
}

type LevelRule struct {
	ID        uint64
	ChatID    int64
	Level     int
	Name      string
	MinPoints int64
	Badge     string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LevelRulePatch struct {
	Level     *int
	Name      *string
	MinPoints *int64
	Badge     *string
	Enabled   *bool
}

type UserLevel struct {
	ChatID       int64
	UserID       int64
	TotalPoints  int64
	Level        int
	Name         string
	Badge        string
	MinPoints    int64
	NextLevel    *LevelRule
	PointsToNext int64
}

type LevelUpgradeResult struct {
	Upgraded      bool
	PreviousLevel UserLevel
	CurrentLevel  UserLevel
}

func (s *LevelService) ListRules(ctx context.Context, chatID int64) ([]LevelRule, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return defaultLevelRules(chatID), nil
	}

	var records []levelRuleRecord
	err := s.store.DB.WithContext(ctx).
		Where("chat_id IN ?", []int64{0, chatID}).
		Order("min_points asc, level asc").
		Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return defaultLevelRules(chatID), nil
		}
		return nil, err
	}
	if len(records) == 0 {
		return defaultLevelRules(chatID), nil
	}

	rules := make([]LevelRule, 0, len(records))
	for _, record := range records {
		rules = append(rules, levelRuleRecordToService(record, chatID))
	}
	return normalizeLevelRules(chatID, rules), nil
}

func (s *LevelService) UpsertRule(ctx context.Context, rule LevelRule) (LevelRule, error) {
	normalizeLevelRule(&rule)
	if s == nil || s.store == nil || s.store.DB == nil {
		rule.UpdatedAt = time.Now()
		if rule.CreatedAt.IsZero() {
			rule.CreatedAt = rule.UpdatedAt
		}
		return rule, nil
	}

	record := serviceLevelRuleToRecord(rule)
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND level = ?", record.ChatID, record.Level).
		Assign(record).
		FirstOrCreate(&record).Error
	if err != nil {
		if isMissingTableError(err) {
			return rule, nil
		}
		return LevelRule{}, err
	}
	return levelRuleRecordToService(record, rule.ChatID), nil
}

func (s *LevelService) UpdateRule(ctx context.Context, chatID int64, level int, patch LevelRulePatch) (LevelRule, error) {
	current := LevelRule{ChatID: chatID, Level: level, Name: defaultLevelName(level), Enabled: true}
	if rules, err := s.ListRules(ctx, chatID); err == nil {
		for _, rule := range rules {
			if rule.Level == level {
				current = rule
				break
			}
		}
	} else if !isMissingTableError(err) {
		return LevelRule{}, err
	}

	applyLevelRulePatch(&current, patch)
	return s.UpsertRule(ctx, current)
}

func (s *LevelService) DeleteRule(ctx context.Context, chatID int64, level int) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND level = ?", chatID, level).
		Delete(&levelRuleRecord{}).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func (s *LevelService) GetUserLevel(ctx context.Context, chatID int64, userID int64) (UserLevel, error) {
	points := int64(0)
	if s != nil && s.store != nil && s.store.DB != nil {
		var point model.UserPoint
		err := s.store.DB.WithContext(ctx).
			Where("chat_id = ? AND user_id = ?", chatID, userID).
			First(&point).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return UserLevel{}, err
		}
		points = point.TotalPoints
	}
	return s.GetUserLevelByPoints(ctx, chatID, userID, points)
}

func (s *LevelService) GetUserLevelByPoints(ctx context.Context, chatID int64, userID int64, points int64) (UserLevel, error) {
	rules, err := s.ListRules(ctx, chatID)
	if err != nil {
		return UserLevel{}, err
	}
	return calculateUserLevel(chatID, userID, points, rules), nil
}

func (s *LevelService) CheckUpgrade(ctx context.Context, chatID int64, userID int64, previousPoints int64, currentPoints int64) (LevelUpgradeResult, error) {
	rules, err := s.ListRules(ctx, chatID)
	if err != nil {
		return LevelUpgradeResult{}, err
	}
	previous := calculateUserLevel(chatID, userID, previousPoints, rules)
	current := calculateUserLevel(chatID, userID, currentPoints, rules)
	return LevelUpgradeResult{
		Upgraded:      current.Level > previous.Level,
		PreviousLevel: previous,
		CurrentLevel:  current,
	}, nil
}

type levelRuleRecord struct {
	model.LevelConfig
}

func (levelRuleRecord) TableName() string {
	return "level_configs"
}

func defaultLevelRules(chatID int64) []LevelRule {
	return []LevelRule{
		{ChatID: chatID, Level: 1, Name: "新手", MinPoints: 0, Badge: "Lv.1", Enabled: true},
		{ChatID: chatID, Level: 2, Name: "活跃", MinPoints: 100, Badge: "Lv.2", Enabled: true},
		{ChatID: chatID, Level: 3, Name: "熟练", MinPoints: 500, Badge: "Lv.3", Enabled: true},
		{ChatID: chatID, Level: 4, Name: "核心", MinPoints: 1000, Badge: "Lv.4", Enabled: true},
		{ChatID: chatID, Level: 5, Name: "传奇", MinPoints: 3000, Badge: "Lv.5", Enabled: true},
	}
}

func normalizeLevelRules(chatID int64, rules []LevelRule) []LevelRule {
	out := make([]LevelRule, 0, len(rules))
	seen := map[int]bool{}
	for _, rule := range rules {
		normalizeLevelRule(&rule)
		if rule.ChatID != chatID && rule.ChatID != 0 {
			continue
		}
		if seen[rule.Level] {
			continue
		}
		seen[rule.Level] = true
		rule.ChatID = chatID
		out = append(out, rule)
	}
	if len(out) == 0 {
		return defaultLevelRules(chatID)
	}
	return out
}

func normalizeLevelRule(rule *LevelRule) {
	if rule.Level <= 0 {
		rule.Level = 1
	}
	if strings.TrimSpace(rule.Name) == "" {
		rule.Name = defaultLevelName(rule.Level)
	}
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Badge = strings.TrimSpace(rule.Badge)
	if rule.Badge == "" {
		rule.Badge = "Lv." + intString(rule.Level)
	}
	if rule.MinPoints < 0 {
		rule.MinPoints = 0
	}
}

func applyLevelRulePatch(rule *LevelRule, patch LevelRulePatch) {
	if patch.Level != nil {
		rule.Level = *patch.Level
	}
	if patch.Name != nil {
		rule.Name = *patch.Name
	}
	if patch.MinPoints != nil {
		rule.MinPoints = *patch.MinPoints
	}
	if patch.Badge != nil {
		rule.Badge = *patch.Badge
	}
	if patch.Enabled != nil {
		rule.Enabled = *patch.Enabled
	}
	normalizeLevelRule(rule)
}

func calculateUserLevel(chatID int64, userID int64, points int64, rules []LevelRule) UserLevel {
	if points < 0 {
		points = 0
	}
	current := rules[0]
	var next *LevelRule
	for i, rule := range rules {
		if points >= rule.MinPoints {
			current = rule
			continue
		}
		next = &rules[i]
		break
	}
	result := UserLevel{
		ChatID:      chatID,
		UserID:      userID,
		TotalPoints: points,
		Level:       current.Level,
		Name:        current.Name,
		Badge:       current.Badge,
		MinPoints:   current.MinPoints,
		NextLevel:   next,
	}
	if next != nil && next.MinPoints > points {
		result.PointsToNext = next.MinPoints - points
	}
	return result
}

func serviceLevelRuleToRecord(rule LevelRule) levelRuleRecord {
	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = now
	}
	return levelRuleRecord{
		LevelConfig: model.LevelConfig{
			ChatID:       rule.ChatID,
			Level:        rule.Level,
			MinPoints:    rule.MinPoints,
			Label:        rule.Name,
			Badge:        rule.Badge,
			CanPostLink:  true,
			CanPostMedia: true,
			BaseModel: model.BaseModel{
				CreatedAt: rule.CreatedAt,
				UpdatedAt: rule.UpdatedAt,
			},
		},
	}
}

func levelRuleRecordToService(record levelRuleRecord, chatID int64) LevelRule {
	name := strings.TrimSpace(record.Label)
	if name == "" {
		name = defaultLevelName(record.Level)
	}
	badge := strings.TrimSpace(record.Badge)
	if badge == "" {
		badge = "Lv." + intString(record.Level)
	}
	return LevelRule{
		ChatID:    chatID,
		Level:     record.Level,
		Name:      name,
		MinPoints: record.MinPoints,
		Badge:     badge,
		Enabled:   true,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func defaultLevelName(level int) string {
	switch level {
	case 1:
		return "新手"
	case 2:
		return "活跃"
	case 3:
		return "熟练"
	case 4:
		return "核心"
	case 5:
		return "传奇"
	default:
		return "等级 " + intString(level)
	}
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "undefined_table")
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "重复键") ||
		strings.Contains(msg, "23505")
}
