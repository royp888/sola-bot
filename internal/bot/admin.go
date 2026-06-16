package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

func (a *App) registerAdminHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("adminconfig", a.wrap(a.handleAdminConfig, a.RateLimit("cmd:adminconfig", 1))))
	d.AddHandler(handlers.NewCommand("set_welcome", a.wrap(a.handleSetWelcome, a.RateLimit("cmd:set_welcome", 1))))
	d.AddHandler(handlers.NewCommand("set_warn_limit", a.wrap(a.handleSetWarnLimit, a.RateLimit("cmd:set_warn_limit", 1))))
	d.AddHandler(handlers.NewCommand("set_level", a.wrap(a.handleSetLevel, a.RateLimit("cmd:set_level", 1))))
	d.AddHandler(handlers.NewCommand("levels", a.wrap(a.handleLevels, a.RateLimit("cmd:levels", 1))))
	d.AddHandler(handlers.NewCommand("add_level", a.wrap(a.handleAddLevel, a.RateLimit("cmd:add_level", 1))))
	d.AddHandler(handlers.NewCommand("del_level", a.wrap(a.handleDelLevel, a.RateLimit("cmd:del_level", 1))))
	a.registerRulesHandlers(d)
}

func (a *App) handleAdminConfig(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	return sendText(b, ctx, formatAdminConfig(cfg), nil)
}

func (a *App) handleSetWelcome(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	text := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	if text == "" {
		return sendText(b, ctx, "用法：/set_welcome 欢迎 {name} 加入！", nil)
	}
	cfg, err := a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID, ChatAdminConfigPatch{WelcomeText: &text})
	if err != nil {
		return err
	}
	return sendText(b, ctx, "欢迎语已更新。\n"+formatAdminConfig(cfg), nil)
}

func (a *App) handleSetWarnLimit(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	args := commandArgs(ctx)
	if len(args) < 1 {
		return sendText(b, ctx, "用法：/set_warn_limit 5", nil)
	}
	limit, err := strconv.Atoi(args[0])
	if err != nil || limit <= 0 {
		return sendText(b, ctx, "警告上限必须是大于 0 的整数。", nil)
	}
	cfg, err := a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID, ChatAdminConfigPatch{WarnLimit: &limit})
	if err != nil {
		return err
	}
	return sendText(b, ctx, "警告上限已更新。\n"+formatAdminConfig(cfg), nil)
}

func (a *App) routeAdminCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "check_admin":
		return a.checkBotAdmin(b, ctx)
	case "config":
		return a.showAdminConfigFromCallback(b, ctx)
	case "verify_menu":
		return a.showVerifyMenu(b, ctx)
	case "verify_toggle":
		return a.toggleVerifyFromCallback(b, ctx)
	case "permissions":
		return a.showPermissionHelp(b, ctx)
	case "moderation":
		return a.showModerationMenu(b, ctx)
	case "mod":
		return a.handleModerationCallback(b, ctx, payload)
	case "keywords":
		return a.showKeywordsFromCallback(b, ctx)
	case "autoreplies":
		return a.showAutoReplyPanel(b, ctx)
	case "bans":
		return a.showBansPanel(b, ctx)
	case "violations":
		return a.showViolationsPanel(b, ctx)
	case "levels":
		return a.showLevelsPanel(b, ctx)
	case "templates":
		return a.showTemplatesPanel(b, ctx)
	case "invites":
		return a.showInvitesPanel(b, ctx)
	default:
		return respondText(b, ctx, "这个入口还在补充更多按钮能力，当前先带你回群组总面板继续操作。", groupMarkup())
	}
}

func (a *App) showAdminConfigFromCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Admin == nil {
		return respondText(b, ctx, "群组配置服务尚未接入。", groupMarkup())
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, formatAdminConfig(cfg), adminConfigMarkup())
}

func (a *App) showPermissionHelp(b *gotgbot.Bot, ctx *ext.Context) error {
	text := strings.Join([]string{
		"🔒 权限控制",
		"",
		"当前版本优先使用 Telegram 原生管理员权限判断。",
		"请在群设置里把机器人设为管理员，并至少授予：删除消息、封禁用户、邀请用户。",
		"",
		"检查当前权限：/check_admin",
	}, "\n")
	return respondText(b, ctx, text, adminConfigMarkup())
}

func (a *App) showKeywordsFromCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.KeywordFilter == nil {
		return respondText(b, ctx, "关键词过滤服务尚未接入。", moderationMenuMarkup())
	}
	text, err := a.services.KeywordFilter.ListKeywords(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	lines := []string{
		"🚫 关键词过滤",
		"━━━━━━━━━━",
		text,
		"",
		"添加：/add_keyword 关键词",
		"删除：/del_keyword 关键词",
		"适合处理广告词、违规词和口令黑名单。",
	}
	return respondText(b, ctx, strings.Join(lines, "\n"), moderationAssetsMarkup())
}

func adminConfigMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "✅ 入群验证", CallbackData: CallbackData("admin", "verify_menu")},
			{Text: "🔑 权限检查", CallbackData: CallbackData("admin", "check_admin")},
		},
		{
			{Text: "🔙 返回群组", CallbackData: CallbackData("menu", "groups")},
		},
	}}}
}

func verifyMenuMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "🔧 开关验证", CallbackData: CallbackData("admin", "verify_toggle")},
			{Text: "⚙️ 群组配置", CallbackData: CallbackData("admin", "config")},
		},
		{
			{Text: "🔙 返回群组", CallbackData: CallbackData("menu", "groups")},
		},
	}}}
}

func deleteMessageLater(b *gotgbot.Bot, chatID int64, messageID int64, delay time.Duration) {
	if delay <= 0 || messageID == 0 {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	<-timer.C
	_, _ = b.DeleteMessageWithContext(context.Background(), chatID, messageID, nil)
}

func (a *App) showLevelsPanel(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.showLevelPanel(b, ctx)
}

func (a *App) showTemplatesPanel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Templates == nil {
		return respondText(b, ctx, "消息模板服务尚未接入。", moderationAssetsMarkup())
	}
	templates, err := a.services.Templates.ListForChat(scope.Context, scope.Chat.ID, 10)
	if err != nil {
		return err
	}
	var builder strings.Builder
	builder.WriteString("📝 消息模板面板\n\n")
	builder.WriteString("常用于欢迎语、活动公告、抽奖文案和客服话术复用。\n")
	if len(templates) == 0 {
		builder.WriteString("当前群还没有消息模板。\n")
	} else {
		builder.WriteString("最近模板：")
		for i, item := range templates {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				name = item.ID
			}
			builder.WriteString(fmt.Sprintf("\n%d. %s [%s]", i+1, truncateRunes(name, 20), templateMediaLabel(item.MediaType)))
			if strings.TrimSpace(item.Content) != "" {
				builder.WriteString(" -> " + truncateRunes(item.Content, 32))
			}
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n快捷命令\n/templates\n/add_template 标题 | 内容\n/del_template 模板ID")
	return respondText(b, ctx, builder.String(), moderationAssetsMarkup())
}

func (a *App) showInvitesPanel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.InviteLink == nil {
		return respondText(b, ctx, "邀请链接服务尚未接入。", moderationAssetsMarkup())
	}
	text, err := a.services.InviteLink.ListForChat(scope.Context, scope.Chat.ID, 20)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		text = "当前群还没有邀请链接记录。"
	}
	panel := strings.Join([]string{
		"🔗 邀请链接面板",
		"",
		"适合区分不同投放入口、合作渠道和拉新来源。",
		text,
		"",
		"快捷命令",
		"/invites",
		"/invite_create 名称",
		"/invite_delete ID 或序号",
	}, "\n")
	return respondText(b, ctx, panel, moderationAssetsMarkup())
}

// Whitelist helpers

func (a *App) postWelcomeMessage(b *gotgbot.Bot, ctx *ext.Context, chatID int64, cfg ChatAdminConfig, user gotgbot.User) error {
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = strconv.FormatInt(user.Id, 10)
	}
	text := strings.ReplaceAll(cfg.WelcomeText, "{name}", name)
	msg, err := b.SendMessageWithContext(requestScope(ctx).Context, chatID, text, nil)
	if err == nil && msg != nil {
		go deleteMessageLater(b, chatID, msg.MessageId, 30*time.Second)
	}
	return err
}

func (a *App) sendWelcomeMessage(b *gotgbot.Bot, ctx *ext.Context, cfg ChatAdminConfig, user gotgbot.User) error {
	return a.postWelcomeMessage(b, ctx, cfg.ChatID, cfg, user)
}

func formatAdminConfig(cfg ChatAdminConfig) string {
	vt := cfg.VerifyType
	if vt == "" {
		vt = "button"
	}
	return fmt.Sprintf(
		"⚙️ 群组管理配置\n适用群组：%d\n欢迎语：%s\n入群验证：%s\n验证类型：%s\n验证超时：%d 秒\n警告上限：%d 次",
		cfg.ChatID,
		cfg.WelcomeText,
		boolLabel(cfg.VerifyEnabled, "开启", "关闭"),
		vt,
		cfg.VerifyTimeout,
		cfg.WarnLimit,
	)
}

func reasonSuffix(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}
	return "原因：" + strings.TrimSpace(reason)
}
