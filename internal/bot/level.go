package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func (a *App) handleSetLevel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Level == nil {
		return sendText(b, ctx, "用户等级服务尚未接入。", nil)
	}

	targetID, level, err := levelTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复用户 /set_level 3，或 /set_level 用户ID 3。", nil)
	}
	if level < 0 {
		return sendText(b, ctx, "等级必须是 >= 0 的整数。", nil)
	}

	result, err := a.services.Level.SetLevel(scope.Context, scope.Chat.ID, targetID, level, scope.Actor.ID)
	if err != nil {
		return err
	}
	result = strings.TrimSpace(result)
	if result == "" {
		result = fmt.Sprintf("已将用户 %d 的等级设置为 %d。", targetID, level)
	}
	return respondText(b, ctx, result, levelPanelMarkup())
}

func levelTarget(ctx *ext.Context) (int64, int, error) {
	args := commandArgs(ctx)
	if len(args) == 0 {
		return 0, 0, fmt.Errorf("missing args")
	}
	if ctx != nil && ctx.Message != nil && ctx.Message.ReplyToMessage != nil && ctx.Message.ReplyToMessage.From != nil {
		level, err := strconv.Atoi(args[0])
		if err != nil {
			return 0, 0, err
		}
		return ctx.Message.ReplyToMessage.From.Id, level, nil
	}
	if len(args) < 2 || strings.HasPrefix(args[0], "@") {
		return 0, 0, fmt.Errorf("missing target")
	}
	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || targetID == 0 {
		return 0, 0, fmt.Errorf("invalid target")
	}
	level, err := strconv.Atoi(args[1])
	if err != nil {
		return 0, 0, err
	}
	return targetID, level, nil
}

func (a *App) showLevelPanel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Level == nil {
		return respondText(b, ctx, "用户等级服务尚未接入。", moderationMenuMarkup())
	}
	text, err := a.services.Level.ListLevelRules(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		text = "当前还没有等级规则。"
	}
	panel := strings.Join([]string{
		"等级规则面板",
		"",
		"成员的活跃积分达到阈值后，可映射为头衔、徽章和分层运营规则。",
		text,
		"",
		"快捷命令",
		"/add_level 2 100 活跃 Lv.2",
		"/del_level 2",
		"/set_level 用户ID 3",
	}, "\n")
	return respondText(b, ctx, panel, levelPanelMarkup())
}

func levelPanelMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "刷新规则", CallbackData: CallbackData("admin", "levels")},
			{Text: "积分中心", CallbackData: CallbackData("points", "menu")},
		},
		{
			{Text: "群组配置", CallbackData: CallbackData("admin", "config")},
			{Text: "返回群管", CallbackData: CallbackData("admin", "moderation")},
		},
		{
			{Text: "返回群组", CallbackData: CallbackData("menu", "group")},
		},
	}}}
}
