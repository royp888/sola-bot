package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

func (a *App) registerRulesHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("setrules", a.wrap(a.handleSetRules, a.RateLimit("cmd:setrules", 1))))
	d.AddHandler(handlers.NewCommand("clearrules", a.wrap(a.handleClearRules, a.RateLimit("cmd:clearrules", 1))))
	d.AddHandler(handlers.NewCommand("rules", a.wrap(a.handleRules, a.RateLimit("cmd:rules", 3))))
}

func (a *App) handleSetRules(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组管理服务尚未接入。", nil)
	}
	var text string
	if ctx.Message != nil && ctx.Message.ReplyToMessage != nil {
		text = strings.TrimSpace(ctx.Message.ReplyToMessage.Text)
	}
	if text == "" {
		text = strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	}
	if text == "" {
		return sendText(b, ctx, "用法：/setrules 群规内容，或回复含群规内容的消息 /setrules。", nil)
	}
	if _, err := a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID, ChatAdminConfigPatch{RulesText: &text}); err != nil {
		return err
	}
	return sendText(b, ctx, "群规已更新。发送 /rules 可查看。", nil)
}

func (a *App) handleClearRules(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组管理服务尚未接入。", nil)
	}
	empty := ""
	if _, err := a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID, ChatAdminConfigPatch{RulesText: &empty}); err != nil {
		return err
	}
	return sendText(b, ctx, "群规已清除。", nil)
}

func (a *App) handleRules(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.Type == "private" {
		return sendText(b, ctx, "请在群组内使用 /rules 查看群规，或点击群内群规按钮。", nil)
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组管理服务尚未接入。", nil)
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.RulesText) == "" {
		return sendText(b, ctx, "本群暂未设置群规。管理员可用 /setrules 设置。", nil)
	}
	botUsername := b.User.Username
	deepLink := fmt.Sprintf("https://t.me/%s?start=rules_%d", botUsername, scope.Chat.ID)
	return sendText(b, ctx, "点击按钮在私聊中查看群规：", &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "📋 查看群规", Url: deepLink}},
		}},
	})
}

func (a *App) handleRulesDeepLink(b *gotgbot.Bot, ctx *ext.Context, param string) error {
	chatIDStr := strings.TrimPrefix(param, "rules_")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil || chatID == 0 {
		return sendText(b, ctx, "无效的群规链接。", nil)
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组管理服务尚未接入。", nil)
	}
	scope := requestScope(ctx)
	cfg, err := a.services.Admin.GetConfig(scope.Context, chatID)
	if err != nil || strings.TrimSpace(cfg.RulesText) == "" {
		return sendText(b, ctx, "该群组暂未设置群规。", nil)
	}
	return sendText(b, ctx, "📋 群规\n━━━━━━━━━━\n"+cfg.RulesText, nil)
}
