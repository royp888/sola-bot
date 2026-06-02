package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

type InviteLinkService struct {
	store *store.Store
	tgBot *gotgbot.Bot
}

func NewInviteLinkService(st *store.Store, token string) *InviteLinkService {
	var tgBot *gotgbot.Bot
	if strings.TrimSpace(token) != "" {
		if bot, err := gotgbot.NewBot(strings.TrimSpace(token), nil); err == nil {
			tgBot = bot
		}
	}
	return &InviteLinkService{store: st, tgBot: tgBot}
}

type InviteLinkListFilter struct {
	ChatID int64
	Limit  int
	Offset int
	Cursor string
}

type InviteLinkCreate struct {
	ChatID             int64
	Name               string
	CreatesJoinRequest bool
	CreatedBy          int64
}

func (s *InviteLinkService) List(ctx context.Context, filter InviteLinkListFilter) ([]model.InviteLink, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []model.InviteLink{}, nil
	}
	db := s.store.DB.WithContext(ctx).Model(&model.InviteLink{})
	if filter.ChatID != 0 {
		db = db.Where("chat_id = ?", filter.ChatID)
	}
	limit := normalLimit(filter.Limit)
	if strings.TrimSpace(filter.Cursor) != "" {
		cursorTime, cursorID, err := decodeUUIDCursor(filter.Cursor)
		if err != nil {
			return nil, err
		}
		db = db.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorTime, cursorTime, cursorID)
	}
	query := db.Order("created_at desc, id desc").Limit(limit)
	if strings.TrimSpace(filter.Cursor) == "" {
		query = query.Offset(filter.Offset)
	}
	var records []model.InviteLink
	err := query.Find(&records).Error
	if err != nil {
		if isMissingTableError(err) {
			return []model.InviteLink{}, nil
		}
		return nil, err
	}
	return records, nil
}

func (s *InviteLinkService) Create(ctx context.Context, req InviteLinkCreate) (model.InviteLink, error) {
	if req.ChatID == 0 {
		return model.InviteLink{}, errors.New("chat_id is required")
	}
	if s == nil || s.tgBot == nil {
		return model.InviteLink{}, errors.New("telegram bot is not configured")
	}
	link, err := s.tgBot.CreateChatInviteLinkWithContext(ctx, req.ChatID, &gotgbot.CreateChatInviteLinkOpts{
		Name:               strings.TrimSpace(req.Name),
		CreatesJoinRequest: req.CreatesJoinRequest,
	})
	if err != nil {
		return model.InviteLink{}, err
	}
	record := model.InviteLink{
		BaseModel:          model.BaseModel{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		ChatID:             req.ChatID,
		Name:               strings.TrimSpace(req.Name),
		InviteLink:         link.InviteLink,
		CreatesJoinRequest: req.CreatesJoinRequest,
		CreatedBy:          req.CreatedBy,
	}
	if s.store == nil || s.store.DB == nil {
		return record, nil
	}
	if err := s.store.DB.WithContext(ctx).Select("*").Create(&record).Error; err != nil {
		return model.InviteLink{}, err
	}
	return record, nil
}

func (s *InviteLinkService) Delete(ctx context.Context, id string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	var record model.InviteLink
	if err := s.store.DB.WithContext(ctx).First(&record, "id = ?", parsed).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || isMissingTableError(err) {
			return nil
		}
		return err
	}
	if s.tgBot != nil && strings.TrimSpace(record.InviteLink) != "" {
		_, _ = s.tgBot.RevokeChatInviteLinkWithContext(ctx, record.ChatID, record.InviteLink, nil)
	}
	err = s.store.DB.WithContext(ctx).Delete(&record).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}

func (s *InviteLinkService) IncrementJoinCount(ctx context.Context, chatID int64, inviteLink string) error {
	if s == nil || s.store == nil || s.store.DB == nil || strings.TrimSpace(inviteLink) == "" {
		return nil
	}
	err := s.store.DB.WithContext(ctx).
		Model(&model.InviteLink{}).
		Where("chat_id = ? AND invite_link = ?", chatID, strings.TrimSpace(inviteLink)).
		UpdateColumn("join_count", gorm.Expr("join_count + 1")).Error
	if isMissingTableError(err) {
		return nil
	}
	return err
}
