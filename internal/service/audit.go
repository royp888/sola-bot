package service

import (
	"context"
	"time"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

// AuditService writes audit log entries to the audit_logs table.
type AuditService struct {
	store *store.Store
}

func NewAuditService(st *store.Store) *AuditService {
	return &AuditService{store: st}
}

// Log writes an audit entry asynchronously so it never blocks the caller.
func (s *AuditService) Log(entry bot.AuditLogEntry) {
	go func() {
		if s == nil || s.store == nil || s.store.DB == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		now := time.Now()
		detail := entry.Detail
		_ = s.store.DB.WithContext(ctx).Create(&model.AuditLog{
			ActorTelegramID:  &entry.ActorTelegramID,
			ChatTelegramID:   &entry.ChatTelegramID,
			Action:           entry.Action,
			EntityType:       entry.EntityType,
			TargetTelegramID: &entry.TargetTelegramID,
			Detail:           &detail,
			OccurredAt:       now,
		}).Error
	}()
}
