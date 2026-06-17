package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dabowin/sola/internal/model"
)

func (s *Store) GetSystemSetting(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.DB == nil {
		return "", false, nil
	}
	var setting model.SystemSetting
	err := s.DB.WithContext(ctx).First(&setting, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		if isSystemSettingsMissingTable(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return setting.Value, true, nil
}

func (s *Store) SetSystemSetting(ctx context.Context, key, value string) error {
	if s == nil || s.DB == nil {
		return nil
	}
	now := time.Now()
	return s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&model.SystemSetting{Key: key, Value: value, UpdatedAt: now}).Error
}

func (s *Store) GetAllSystemSettings(ctx context.Context) (map[string]string, error) {
	if s == nil || s.DB == nil {
		return map[string]string{}, nil
	}
	var settings []model.SystemSetting
	if err := s.DB.WithContext(ctx).Find(&settings).Error; err != nil {
		if isSystemSettingsMissingTable(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := make(map[string]string, len(settings))
	for _, setting := range settings {
		out[setting.Key] = setting.Value
	}
	return out, nil
}

func isSystemSettingsMissingTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return (strings.Contains(msg, "relation") && strings.Contains(msg, "does not exist")) ||
		strings.Contains(msg, "no such table")
}
