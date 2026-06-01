package bot

import (
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/dabowin/sola/internal/api"
)

func (a *App) handleBind(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.checkBotAdmin(b, ctx)
}

func (a *App) handleCheckAdmin(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.checkBotAdmin(b, ctx)
}

func (a *App) checkBotAdmin(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.ID == 0 {
		return respondText(b, ctx, "没有检测到当前聊天。请在目标群组或频道里发送 /bind 或 /check_admin。", backHomeMarkup())
	}
	if scope.Chat.Type == "private" {
		return respondText(b, ctx, "请先把我添加到目标群组/频道，并授予管理员权限，然后在目标聊天里发送 /bind 或 /check_admin。", backHomeMarkup())
	}
	if a.services.TelegramAccess == nil {
		return respondText(b, ctx, "Telegram 权限检查服务尚未接入。", backHomeMarkup())
	}

	status, err := a.services.TelegramAccess.CheckBotAdmin(scope.Context, b, scope.Chat.ID)
	if err != nil {
		return err
	}
	userStatus, err := a.services.TelegramAccess.CheckUserAdmin(scope.Context, b, scope.Chat.ID, scope.Actor.ID)
	if err != nil {
		return err
	}
	if !userStatus.IsAdmin && !userStatus.CanManageChat {
		return respondText(b, ctx, "只有群主或管理员可以绑定这个群。请用群管理员账号发送 /bind。", groupMarkup())
	}
	if status.IsAdmin && a.services.ChatBindings != nil {
		if _, err := a.services.ChatBindings.Bind(scope.Context, api.ChatBindingRequest{
			ChatID:              scope.Chat.ID,
			ChatType:            scope.Chat.Type,
			Title:               scope.Chat.Title,
			Username:            scope.Chat.Username,
			BoundBy:             actorLabel(scope.Actor),
			OwnerTelegramUserID: scope.Actor.ID,
			OwnerUsername:       scope.Actor.Username,
			OwnerDisplayName:    actorLabel(scope.Actor),
		}); err != nil {
			return respondText(b, ctx, fmt.Sprintf("绑定失败：%s", err.Error()), groupMarkup())
		}
	}
	if status.IsAdmin {
		return respondText(b, ctx, formatBotAdminStatus(scope.Chat, status)+"\n\n绑定成功：现在你可以用 Telegram 账号登录后台管理这个群。", groupMarkup())
	}
	return respondText(b, ctx, formatBotAdminStatus(scope.Chat, status), groupMarkup())
}

func actorLabel(actor Actor) string {
	if strings.TrimSpace(actor.Username) != "" {
		return "@" + strings.TrimSpace(actor.Username)
	}
	if strings.TrimSpace(actor.FirstName) != "" {
		return strings.TrimSpace(actor.FirstName)
	}
	if actor.ID != 0 {
		return fmt.Sprintf("%d", actor.ID)
	}
	return ""
}

func formatBotAdminStatus(chat ChatRef, status BotAdminStatus) string {
	title := chat.Title
	if title == "" {
		title = fmt.Sprintf("%d", chat.ID)
	}

	var builder strings.Builder
	builder.WriteString("绑定权限检查\n")
	builder.WriteString(fmt.Sprintf("聊天：%s (%s)\n", title, chat.Type))
	builder.WriteString(fmt.Sprintf("Chat ID：%d\n", chat.ID))
	builder.WriteString(fmt.Sprintf("Bot ID：%d\n", status.BotID))
	builder.WriteString(fmt.Sprintf("Telegram 状态：%s\n\n", status.Status))

	if !status.IsAdmin {
		builder.WriteString("结果：Bot 还不是管理员。\n")
		builder.WriteString("请在 Telegram 里把机器人设为管理员，再重新执行 /bind。")
		return builder.String()
	}

	builder.WriteString("结果：Bot 已是管理员，可以继续绑定。\n")
	builder.WriteString("权限：")
	builder.WriteString(adminPermissionLine(status))
	return builder.String()
}

func adminPermissionLine(status BotAdminStatus) string {
	permissions := []string{}
	if status.CanManageChat {
		permissions = append(permissions, "管理聊天")
	}
	if status.CanPostMessages {
		permissions = append(permissions, "发布消息")
	}
	if status.CanEditMessages {
		permissions = append(permissions, "编辑消息")
	}
	if status.CanDeleteMessages {
		permissions = append(permissions, "删除消息")
	}
	if status.CanRestrictMembers {
		permissions = append(permissions, "限制成员")
	}
	if status.CanInviteUsers {
		permissions = append(permissions, "邀请用户")
	}
	if status.CanPinMessages {
		permissions = append(permissions, "置顶消息")
	}
	if status.CanPromoteMembers {
		permissions = append(permissions, "提升管理员")
	}
	if status.CanManageTopics {
		permissions = append(permissions, "管理话题")
	}
	if status.CanManageDirectMessages {
		permissions = append(permissions, "管理频道私信")
	}
	if len(permissions) == 0 {
		return "管理员身份已确认，Telegram 未返回细分权限。"
	}
	return strings.Join(permissions, " / ")
}
