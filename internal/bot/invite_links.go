package bot

import (
	"context"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func (a *App) handleChatMemberInviteLink(_ *gotgbot.Bot, ctx *ext.Context) error {
	if ctx == nil || ctx.ChatMember == nil || ctx.ChatMember.InviteLink == nil || a.services.InviteLink == nil {
		return nil
	}
	status := ctx.ChatMember.NewChatMember.GetStatus()
	if status != "member" && status != "administrator" && status != "creator" {
		return nil
	}
	return a.services.InviteLink.IncrementJoinCount(context.Background(), ctx.ChatMember.Chat.Id, ctx.ChatMember.InviteLink.InviteLink)
}
