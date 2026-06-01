package service

import (
	"context"
	"fmt"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/dabowin/sola/internal/bot"
)

type TelegramAccessService struct{}

const telegramAccessTimeout = 15 * time.Second

func NewTelegramAccessService() *TelegramAccessService { return &TelegramAccessService{} }

func (s *TelegramAccessService) CheckBotAdmin(ctx context.Context, tgBot *gotgbot.Bot, chatID int64) (bot.BotAdminStatus, error) {
	if tgBot == nil {
		return bot.BotAdminStatus{}, fmt.Errorf("telegram bot is nil")
	}

	botID := tgBot.User.Id
	if botID == 0 {
		reqCtx, cancel := context.WithTimeout(ctx, telegramAccessTimeout)
		defer cancel()
		me, err := tgBot.GetMeWithContext(reqCtx, nil)
		if err != nil {
			return bot.BotAdminStatus{}, err
		}
		botID = me.Id
	}

	return s.checkMemberAdmin(ctx, tgBot, chatID, botID)
}

func (s *TelegramAccessService) CheckUserAdmin(ctx context.Context, tgBot *gotgbot.Bot, chatID int64, userID int64) (bot.BotAdminStatus, error) {
	if tgBot == nil {
		return bot.BotAdminStatus{}, fmt.Errorf("telegram bot is nil")
	}
	if userID == 0 {
		return bot.BotAdminStatus{}, fmt.Errorf("telegram user id is empty")
	}
	return s.checkMemberAdmin(ctx, tgBot, chatID, userID)
}

func (s *TelegramAccessService) checkMemberAdmin(ctx context.Context, tgBot *gotgbot.Bot, chatID int64, userID int64) (bot.BotAdminStatus, error) {
	reqCtx, cancel := context.WithTimeout(ctx, telegramAccessTimeout)
	defer cancel()
	member, err := tgBot.GetChatMemberWithContext(reqCtx, chatID, userID, nil)
	if err != nil {
		return bot.BotAdminStatus{}, err
	}

	status := bot.BotAdminStatus{
		ChatID: chatID,
		BotID:  userID,
		Status: member.GetStatus(),
	}

	switch m := member.(type) {
	case gotgbot.ChatMemberOwner:
		status.IsAdmin = true
		status.CanManageChat = true
		status.CanPostMessages = true
		status.CanEditMessages = true
		status.CanDeleteMessages = true
		status.CanRestrictMembers = true
		status.CanInviteUsers = true
		status.CanPinMessages = true
		status.CanPromoteMembers = true
		status.CanManageTopics = true
		status.CanManageDirectMessages = true
	case gotgbot.ChatMemberAdministrator:
		status.IsAdmin = true
		status.CanManageChat = m.CanManageChat
		status.CanPostMessages = m.CanPostMessages
		status.CanEditMessages = m.CanEditMessages
		status.CanDeleteMessages = m.CanDeleteMessages
		status.CanRestrictMembers = m.CanRestrictMembers
		status.CanInviteUsers = m.CanInviteUsers
		status.CanPinMessages = m.CanPinMessages
		status.CanPromoteMembers = m.CanPromoteMembers
		status.CanManageTopics = m.CanManageTopics
		status.CanManageDirectMessages = m.CanManageDirectMessages
	default:
		status.IsAdmin = false
	}

	return status, nil
}
