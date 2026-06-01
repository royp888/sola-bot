package store

import (
	"context"

	"github.com/dabowin/sola/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository[T any] struct {
	db *gorm.DB
}

func newRepository[T any](db *gorm.DB) repository[T] {
	return repository[T]{db: db}
}

func (r repository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r repository[T]) GetByID(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T
	if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r repository[T]) List(ctx context.Context, limit, offset int) ([]T, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	entities := make([]T, 0, limit)
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&entities).Error
	return entities, err
}

func (r repository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r repository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(new(T), "id = ?", id).Error
}

type UserRepository struct{ repository[model.User] }
type BotRepository struct{ repository[model.Bot] }
type TelegramChatRepository struct{ repository[model.TelegramChat] }
type ChatAdminRepository struct{ repository[model.ChatAdmin] }
type PostRepository struct{ repository[model.Post] }
type ScheduledJobRepository struct{ repository[model.ScheduledJob] }
type ButtonTemplateRepository struct {
	repository[model.ButtonTemplate]
}
type EventRepository struct{ repository[model.Event] }
type PointRepository struct{ repository[model.Point] }
type AuditLogRepository struct{ repository[model.AuditLog] }

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{repository: newRepository[model.User](db)}
}

func NewBotRepository(db *gorm.DB) *BotRepository {
	return &BotRepository{repository: newRepository[model.Bot](db)}
}

func NewTelegramChatRepository(db *gorm.DB) *TelegramChatRepository {
	return &TelegramChatRepository{repository: newRepository[model.TelegramChat](db)}
}

func NewChatAdminRepository(db *gorm.DB) *ChatAdminRepository {
	return &ChatAdminRepository{repository: newRepository[model.ChatAdmin](db)}
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{repository: newRepository[model.Post](db)}
}

func NewScheduledJobRepository(db *gorm.DB) *ScheduledJobRepository {
	return &ScheduledJobRepository{repository: newRepository[model.ScheduledJob](db)}
}

func NewButtonTemplateRepository(db *gorm.DB) *ButtonTemplateRepository {
	return &ButtonTemplateRepository{repository: newRepository[model.ButtonTemplate](db)}
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{repository: newRepository[model.Event](db)}
}

func NewPointRepository(db *gorm.DB) *PointRepository {
	return &PointRepository{repository: newRepository[model.Point](db)}
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{repository: newRepository[model.AuditLog](db)}
}
