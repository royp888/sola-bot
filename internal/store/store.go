package store

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/model"
)

type Store struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func New(db *gorm.DB, redis *redis.Client) *Store {
	return &Store{DB: db, Redis: redis}
}

func (s *Store) AutoMigrate(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.WithContext(ctx).AutoMigrate(
		&model.User{},
		&model.Bot{},
		&model.TelegramChat{},
		&model.ChatPointConfig{},
		&model.UserPoint{},
		&model.PointLog{},
		&model.BanLog{},
		&model.WarnRecord{},
		&model.ChatAdminConfig{},
		&model.ChatAdmin{},
		&model.Post{},
		&model.ScheduledPost{},
		&model.ScheduledPostDelivery{},
		&model.ScheduledJob{},
		&model.MessageTemplate{},
		&model.ButtonTemplate{},
		&model.Event{},
		&model.Point{},
		&model.Lottery{},
		&model.LotteryEntry{},
		&model.LevelConfig{},
		&model.ChatModerationConfig{},
		&model.KeywordFilter{},
		&model.AutoReply{},
		&model.ViolationRecord{},
		&model.InviteLink{},
		&model.AuditLog{},
		&model.SeenUser{},
	)
}
