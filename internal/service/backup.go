package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/store"
)

type BackupService struct {
	store *store.Store
}

func NewBackupService(st *store.Store) *BackupService {
	return &BackupService{store: st}
}

type BackupData struct {
	Version    string                     `json:"version"`
	ExportedAt string                     `json:"exported_at"`
	Scope      string                     `json:"scope"`
	Tables     map[string]json.RawMessage `json:"tables"`
}

var businessBackupTables = []string{
	"chat_point_configs",
	"user_points",
	"point_logs",
	"level_configs",
	"lotteries",
	"lottery_entries",
	"chat_moderation_configs",
	"keyword_filters",
	"auto_replies",
	"message_templates",
	"scheduled_posts",
	"invite_links",
}

var fullBackupExtraTables = []string{
	"violation_records",
	"ban_logs",
	"warn_records",
	"telegram_chats",
	"bots",
	"users",
	"chat_admins",
	"scheduled_jobs",
	"events",
	"audit_logs",
}

func (s *BackupService) Export(ctx context.Context, scope string) (*BackupData, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	scope = normalizeBackupScope(scope)
	data := &BackupData{
		Version:    "1",
		ExportedAt: time.Now().Format(time.RFC3339),
		Scope:      scope,
		Tables:     map[string]json.RawMessage{},
	}
	for _, table := range backupTablesForScope(scope) {
		var rows []map[string]any
		err := s.store.DB.WithContext(ctx).Table(table).Find(&rows).Error
		if err != nil {
			if isMissingTableError(err) {
				continue
			}
			return nil, fmt.Errorf("export %s: %w", table, err)
		}
		raw, err := json.Marshal(rows)
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", table, err)
		}
		data.Tables[table] = raw
	}
	return data, nil
}

func (s *BackupService) Import(ctx context.Context, data *BackupData, mode string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return gorm.ErrInvalidDB
	}
	if data == nil || len(data.Tables) == 0 {
		return errors.New("backup data is empty")
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "merge"
	}
	if mode != "merge" && mode != "overwrite" {
		return errors.New("mode must be merge or overwrite")
	}
	allowed := allowedBackupTableSet()
	for table := range data.Tables {
		if !allowed[table] {
			return fmt.Errorf("table %s is not allowed", table)
		}
	}

	return s.store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ordered := backupTablesForScope("full")
		for _, table := range ordered {
			raw, ok := data.Tables[table]
			if !ok {
				continue
			}
			var rows []map[string]any
			if err := json.Unmarshal(raw, &rows); err != nil {
				return fmt.Errorf("decode %s: %w", table, err)
			}
			if mode == "overwrite" {
				if err := tx.Exec("DELETE FROM " + table).Error; err != nil {
					if isMissingTableError(err) {
						continue
					}
					return fmt.Errorf("clear %s: %w", table, err)
				}
			}
			for start := 0; start < len(rows); start += 500 {
				end := start + 500
				if end > len(rows) {
					end = len(rows)
				}
				if err := tx.Table(table).Create(rows[start:end]).Error; err != nil {
					if isMissingTableError(err) {
						continue
					}
					return fmt.Errorf("import %s: %w", table, err)
				}
			}
		}
		return nil
	})
}

func backupTablesForScope(scope string) []string {
	tables := append([]string{}, businessBackupTables...)
	if normalizeBackupScope(scope) == "full" {
		tables = append(tables, fullBackupExtraTables...)
	}
	return tables
}

func allowedBackupTableSet() map[string]bool {
	allowed := map[string]bool{}
	for _, table := range backupTablesForScope("full") {
		allowed[table] = true
	}
	return allowed
}

func normalizeBackupScope(scope string) string {
	if strings.EqualFold(strings.TrimSpace(scope), "full") {
		return "full"
	}
	return "business"
}
