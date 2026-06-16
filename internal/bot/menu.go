package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
)

func (a *App) registerCoreHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("start", a.wrap(a.handleStart, a.RateLimit("cmd:start", 1))))
	d.AddHandler(handlers.NewCommand("menu", a.wrap(a.handleStart, a.RateLimit("cmd:menu", 1))))
	d.AddHandler(handlers.NewCommand("help", a.wrap(a.handleHelp, a.RateLimit("cmd:help", 1))))
	d.AddHandler(handlers.NewCommand("info", a.wrap(a.handleInfo, a.RateLimit("cmd:info", 1))))
	d.AddHandler(handlers.NewCommand("html", a.wrap(a.handleHTMLHelp, a.RateLimit("cmd:html", 1))))
	d.AddHandler(handlers.NewCommand("bind", a.wrap(a.handleBind, a.RateLimit("cmd:bind", 1))))
	d.AddHandler(handlers.NewCommand("check_admin", a.wrap(a.handleCheckAdmin, a.RateLimit("cmd:check_admin", 1))))
	d.AddHandler(handlers.NewCommand("cancel", a.wrap(a.handleCancel, a.RateLimit("cmd:cancel", 1))))
	d.AddHandler(handlers.NewMessage(message.All, a.handleChineseCommand))
	a.registerSedHandlers(d)
	d.AddHandler(handlers.NewCallback(callbackquery.Prefix(CallbackPrefix+":"), a.wrap(a.router.Handle, a.RateLimit("callback", 1))))
}

func (a *App) handleStart(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message != nil {
		args := commandArgs(ctx)
		if len(args) > 0 && strings.HasPrefix(args[0], "rules_") {
			return a.handleRulesDeepLink(b, ctx, args[0])
		}
	}
	return a.showHomeMenu(b, ctx)
}

func (a *App) handleHelp(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.Type == "private" {
		return sendText(b, ctx, privateHelpText(), nil)
	}
	isAdmin := false
	if a.services.TelegramAccess != nil && scope.Actor.ID != 0 {
		status, err := a.services.TelegramAccess.CheckUserAdmin(scope.Context, b, scope.Chat.ID, scope.Actor.ID)
		isAdmin = err == nil && status.IsAdmin
	}
	if isAdmin {
		return sendText(b, ctx, groupAdminHelpText(), nil)
	}
	return sendText(b, ctx, groupMemberHelpText(), nil)
}

func (a *App) handleHTMLHelp(b *gotgbot.Bot, ctx *ext.Context) error {
	text := strings.Join([]string{
		"<b>如何使用 HTML?</b>",
		"",
		"&lt;b&gt;加粗&lt;/b&gt; <b>加粗</b>",
		"&lt;i&gt;斜体&lt;/i&gt; <i>斜体</i>",
		"&lt;u&gt;下划线&lt;/u&gt; <u>下划线</u>",
		"&lt;s&gt;删除线&lt;/s&gt; <s>删除线</s>",
		"",
		"&lt;a href=&quot;https://telegram.org&quot;&gt;链接&lt;/a&gt; <a href=\"https://telegram.org\">链接</a>",
		"&lt;code&gt;点击复制文本&lt;/code&gt; <code>点击复制文本</code>",
		"",
		"&lt;pre&gt;文本块&lt;/pre&gt;",
		"<pre>我是文本块</pre>",
		"",
		"&lt;blockquote&gt;块引用&lt;/blockquote&gt;",
		"<blockquote>我是块引用</blockquote>",
	}, "\n")
	return sendText(b, ctx, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"})
}

func (a *App) handleInfo(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.ID == 0 {
		return sendText(b, ctx, "没有获取到当前会话信息。", nil)
	}

	var lines []string
	lines = append(lines,
		"当前会话信息",
		fmt.Sprintf("Chat ID：%d", scope.Chat.ID),
		fmt.Sprintf("类型：%s", scope.Chat.Type),
	)
	if strings.TrimSpace(scope.Chat.Title) != "" {
		lines = append(lines, "标题："+scope.Chat.Title)
	}
	if scope.Actor.ID != 0 {
		name := strings.TrimSpace(scope.Actor.FirstName)
		if scope.Actor.Username != "" {
			name = strings.TrimSpace(name + " @" + scope.Actor.Username)
		}
		lines = append(lines, fmt.Sprintf("你：%s (%d)", name, scope.Actor.ID))
	}
	if target := infoTarget(ctx); target != nil {
		name := strings.TrimSpace(target.FirstName + " " + target.LastName)
		if target.Username != "" {
			name = strings.TrimSpace(name + " @" + target.Username)
		}
		if name == "" {
			name = strconv.FormatInt(target.Id, 10)
		}
		lines = append(lines, fmt.Sprintf("目标用户：%s (%d)", name, target.Id))
	}
	if a.services.TelegramAccess != nil && (scope.Chat.Type == "group" || scope.Chat.Type == "supergroup" || scope.Chat.Type == "channel") {
		status, err := a.services.TelegramAccess.CheckBotAdmin(scope.Context, b, scope.Chat.ID)
		if err != nil {
			lines = append(lines, "Bot 权限：检查失败："+err.Error())
		} else if status.IsAdmin {
			lines = append(lines, fmt.Sprintf("Bot 权限：管理员，禁言=%t 删除=%t 发帖=%t", status.CanRestrictMembers, status.CanDeleteMessages, status.CanPostMessages))
		} else {
			lines = append(lines, "Bot 权限：不是管理员")
		}
	}
	return sendText(b, ctx, strings.Join(lines, "\n"), nil)
}

func privateHelpText() string {
	return strings.Join([]string{
		"私聊功能说明",
		"",
		"/start 打开运营控制台",
		"/help 查看私聊功能",
		"/html 查看 HTML 格式示例",
		"/info 查看你的用户信息",
		"",
		"常用入口",
		"1. 先在私聊里选择要管理的群组或频道",
		"2. 进入当前工作台后，可点抽奖、积分中心、定时发帖、群管中心",
		"3. 成员级操作建议回到群里回复目标消息执行",
		"",
		"频道/后台",
		"/bind 绑定频道或群组",
		"/check_admin 检查 Bot 管理员权限",
		"/publish 打开快捷发布入口",
		"/posts 查看定时发帖任务",
		"",
		"群组功能请在群里发送 /help。",
	}, "\n")
}

func groupMemberHelpText() string {
	return strings.Join([]string{
		"群组成员命令",
		"",
		"/points 查看自己的积分",
		"/rank 查看积分排行榜（/rank day 今日 /rank week 本周）",
		"/sign 每日签到",
		"/lottery 查看进行中的抽奖",
		"/info 查看当前群/用户信息",
		"/help 查看本说明",
		"",
		"管理员功能请联系群主。",
	}, "\n")
}

func groupAdminHelpText() string {
	return strings.Join([]string{
		"管理员命令",
		"",
		"-- 成员管理 --",
		"回复消息 /manage 打开按钮管理面板",
		"/ban 原因 封禁（回复消息或 /ban 用户ID 原因）",
		"/unban 用户ID 解封",
		"/mute 30m 禁言（30m/2h/1d）",
		"/unmute 解除禁言",
		"/kick 踢出",
		"/warn 原因 警告  /unwarn 清除  /warns 查看",
		"/bans 封禁记录  /violations 违规记录",
		"",
		"-- 积分 --",
		"回复 /points 10 给用户加减积分",
		"/points_config 查看积分配置",
		"/points_toggle 开关积分系统",
		"",
		"-- 运营 --",
		"/publish 内容 立即发布",
		"/posts 定时任务列表",
		"/post_create 创建定时提醒/循环发布",
		"",
		"-- 设置 --",
		"/adminconfig 查看群组配置",
		"/set_welcome 文本 设置欢迎语（支持 {name}）",
		"/verify_toggle 开关入群验证",
		"/keywords 关键词规则",
		"/invites 邀请链接管理",
		"/levels 等级规则",
	}, "\n")
}

func infoTarget(ctx *ext.Context) *gotgbot.User {
	if ctx == nil || ctx.Message == nil {
		return nil
	}
	if ctx.Message.ReplyToMessage != nil && ctx.Message.ReplyToMessage.From != nil {
		return ctx.Message.ReplyToMessage.From
	}
	for _, entity := range ctx.Message.GetEntities() {
		if entity.Type == "text_mention" && entity.User != nil {
			return entity.User
		}
	}
	return nil
}

func escapedHTML(raw string) string {
	return html.EscapeString(raw)
}

func (a *App) routeMenuCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "home":
		return a.showHomeMenu(b, ctx)
	case "channels":
		return a.showChannelMenu(b, ctx)
	case "groups":
		return a.showGroupMenu(b, ctx)
	case "private_help":
		return respondText(b, ctx, privateHelpText(), backPrivateHomeMarkup())
	case "noop":
		return answerCallback(b, ctx, "")
	default:
		return respondText(b, ctx, "这个菜单项还在开发中。", backHomeMarkup())
	}
}

func (a *App) showHomeMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.Type == "group" || scope.Chat.Type == "supergroup" {
		return a.showGroupMenu(b, ctx)
	}
	if scope.Chat.Type == "channel" {
		return a.showChannelMenu(b, ctx)
	}
	return a.showPrivateHome(b, ctx)
}

func (a *App) showChannelMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	return respondText(b, ctx, "📢 频道设置\n━━━━━━━━━━\n选择要更改的项目：", channelMarkup())
}

func (a *App) showGroupMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	lines := []string{
		"🎛 群组控制面板",
		"━━━━━━━━━━",
	}
	if strings.TrimSpace(scope.Chat.Title) != "" {
		lines = append(lines, fmt.Sprintf("📍 当前群组：%s", scope.Chat.Title))
	}
	lines = append(lines,
		fmt.Sprintf("🆔 Chat ID：%d", scope.Chat.ID),
		"",
		"这里是群内成员和管理员共用的快捷入口。",
		"成员可直接查看积分、排行榜、抽奖活动；管理员可继续进入群管、发帖、验证和关键词管理。\n\n📱 点击下方按钮打开手机控制台（Mini App）。",
	)
	return respondText(b, ctx, strings.Join(lines, "\n"), groupMarkup())
}

func channelMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "⏰ 定时消息", CallbackData: CallbackData("publish", "queue")},
			{Text: "📣 快捷发布", CallbackData: CallbackData("publish", "quick")},
		},
		{
			{Text: "🔗 检查绑定", CallbackData: CallbackData("admin", "check_admin")},
		},
		{
			{Text: "🔄 频道同步", CallbackData: CallbackData("publish", "sync")},
			{Text: "🔒 控制权限", CallbackData: CallbackData("admin", "permissions")},
		},
		{
			{Text: "📊 数据统计", CallbackData: CallbackData("points", "stats")},
			{Text: "🏠 首页", CallbackData: CallbackData("menu", "home")},
		},
	}}}
}

func groupMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "💎 我的积分", CallbackData: CallbackData("points", "menu")},
			{Text: "🎁 抽奖大厅", CallbackData: CallbackData("lottery", "active")},
		},
		{
			{Text: "📣 发布中心", CallbackData: CallbackData("publish", "quick")},
			{Text: "⚙️ 群组配置", CallbackData: CallbackData("admin", "config")},
		},
		{
			{Text: "🛡 群管中心", CallbackData: CallbackData("admin", "moderation")},
			{Text: "🔍 关键词", CallbackData: CallbackData("admin", "keywords")},
		},
		{
			{Text: "🏅 等级规则", CallbackData: CallbackData("admin", "levels")},
			{Text: "🔗 邀请链接", CallbackData: CallbackData("admin", "invites")},
		},
		{
			{Text: "📋 私聊工作台", CallbackData: CallbackData("private", "home")},
		},
	}}}
}

func backHomeMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "🔙 返回首页", CallbackData: CallbackData("menu", "home")}},
	}}}
}

func backPrivateHomeMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "🔙 返回运营台", CallbackData: CallbackData("private", "home")}},
	}}}
}
