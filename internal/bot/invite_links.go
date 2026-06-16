package bot

import (
	"context"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/chatmember"
)

func (a *App) registerInviteHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("invites", a.wrap(a.handleInvites, a.RateLimit("cmd:invites", 1))))
	d.AddHandler(handlers.NewCommand("invite_create", a.wrap(a.handleInviteCreate, a.RateLimit("cmd:invite_create", 1))))
	d.AddHandler(handlers.NewCommand("invite_delete", a.wrap(a.handleInviteDelete, a.RateLimit("cmd:invite_delete", 1))))
	d.AddHandler(handlers.NewChatMember(chatmember.InviteLink, a.handleChatMemberInviteLink))
}

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
