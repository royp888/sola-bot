package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func (a *App) handleStart(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.showHomeMenu(b, ctx)
}

func (a *App) handleHelp(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.Type == "private" {
		return sendText(b, ctx, privateHelpText(), nil)
	}
	return sendText(b, ctx, groupHelpText(), nil)
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

func groupHelpText() string {
	return strings.Join([]string{
		"群组管理命令说明",
		"",
		"基础设置",
		"/start 打开群组设置菜单",
		"/bind 绑定当前群组",
		"/check_admin 检查 Bot 管理员权限",
		"/adminconfig 查看群组配置",
		"/set_welcome 欢迎 {name} 加入！",
		"/set_warn_limit 5 设置警告上限",
		"/set_level 用户ID 3 设置用户等级",
		"/levels 查看等级规则",
		"/add_level 2 100 活跃 Lv.2 添加或修改等级规则",
		"/del_level 2 删除等级规则",
		"/verify_toggle 开关入群验证",
		"",
		"成员管理",
		"回复用户 /manage 打开按钮管理面板",
		"/bans 查看封禁记录",
		"/mute 禁言：回复用户 /mute 30m，或 /mute 用户ID 30m",
		"/unmute 解除禁言：回复用户 /unmute，或 /unmute 用户ID",
		"/ban 封禁：回复用户 /ban 原因，或 /ban 用户ID 原因",
		"/unban 解封：/unban 用户ID",
		"/kick 踢出成员：回复用户 /kick，或 /kick 用户ID",
		"/warn 警告：回复用户 /warn 原因，或 /warn 用户ID 原因",
		"/unwarn 清除警告：回复用户 /unwarn，或 /unwarn 用户ID",
		"/warns 查看警告：回复用户 /warns，或 /warns 用户ID",
		"/violations 查看违规记录",
		"/resolve_violation ID 备注 处理违规记录",
		"/ignore_violation ID 备注 忽略违规记录",
		"",
		"用户信息",
		"/info 查看当前群信息",
		"回复用户 /info 查看被回复用户信息",
		"",
		"积分系统",
		"/points 查看自己的积分",
		"回复用户 /points 10 给该用户增加 10 积分",
		"回复用户 /points -10 给该用户扣除 10 积分",
		"/points 用户ID 10 给指定用户增加 10 积分",
		"/rank 或 /points_rank 查看总积分榜",
		"/rank day 今日榜，/rank week 本周榜，/rank month 本月榜",
		"/points_config 查看积分配置",
		"/set_points text 2 设置文字消息分值",
		"/set_points photo 5 设置图片消息分值",
		"/set_cooldown 30 设置防刷间隔",
		"/points_toggle 开关积分系统",
		"",
		"关键词过滤",
		"/keywords 查看过滤关键词",
		"/add_keyword 广告 添加过滤关键词",
		"/del_keyword 广告 删除过滤关键词",
		"/replies 查看自动回复",
		"/add_reply 关键词 | 回复内容 添加自动回复",
		"/del_reply 关键词 删除自动回复",
		"/templates 查看消息模板",
		"/add_template 标题 | 内容 添加消息模板",
		"/del_template ID 删除消息模板",
		"",
		"运营功能",
		"/publish 要发布的内容 立即发布",
		"/posts 查看本群定时任务",
		"/post_create 创建定时提醒/循环发布",
		"/post_toggle ID 开关定时任务",
		"/post_delete ID 删除定时任务",
		"/invites 查看邀请链接",
		"/invite_create 名称 创建邀请链接",
		"/invite_delete ID 删除邀请链接",
		"/lottery list 查看进行中的抽奖",
		"/html 查看发布 HTML 示例",
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
	case "language":
		return a.showLanguageMenu(b, ctx)
	case "timezone":
		return a.showTimezoneMenu(b, ctx)
	case "private_help":
		return respondText(b, ctx, privateHelpText(), backPrivateHomeMarkup())
	case "noop":
		return answerCallback(b, ctx, "当前版本已使用中文")
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

func (a *App) showLanguageMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	text := strings.Join([]string{
		"🌐 语言设置",
		"━━━━━━━━━━",
		"",
		"当前：简体中文",
		"后台与 Bot 文案目前默认中文，English 菜单已保留入口。",
	}, "\n")
	return respondText(b, ctx, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "简体中文", CallbackData: CallbackData("menu", "noop")},
				{Text: "English", CallbackData: CallbackData("menu", "noop")},
			},
			{
				{Text: "🔙 返回首页", CallbackData: CallbackData("menu", "home")},
			},
		}},
	})
}

func (a *App) showTimezoneMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	text := strings.Join([]string{
		"🕐 时区设置",
		"━━━━━━━━━━",
		"",
		"当前调度按服务器本地时间运行。后台创建定时任务时，请直接用页面里的日期/时间选择器。",
		"常用时区：UTC+8 北京/上海，UTC+0 伦敦，UTC-5 美东。",
	}, "\n")
	return respondText(b, ctx, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "UTC+8", CallbackData: CallbackData("menu", "noop")},
				{Text: "UTC+0", CallbackData: CallbackData("menu", "noop")},
				{Text: "UTC-5", CallbackData: CallbackData("menu", "noop")},
			},
			{
				{Text: "🔙 返回首页", CallbackData: CallbackData("menu", "home")},
			},
		}},
	})
}

func homeMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "👥 群组入口", CallbackData: CallbackData("menu", "groups")},
			{Text: "📢 频道入口", CallbackData: CallbackData("menu", "channels")},
		},
		{
			{Text: "📋 私聊工作台", CallbackData: CallbackData("private", "home")},
			{Text: "🔑 检查权限", CallbackData: CallbackData("admin", "check_admin")},
		},
		{
			{Text: "📣 快捷发布", CallbackData: CallbackData("publish", "quick")},
			{Text: "🎁 抽奖大厅", CallbackData: CallbackData("lottery", "active")},
		},
		{
			{Text: "🕐 时区设置", CallbackData: CallbackData("menu", "timezone")},
			{Text: "🌐 语言设置", CallbackData: CallbackData("menu", "language")},
		},
	}}}
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
			{Text: "📣 定时发帖", CallbackData: CallbackData("publish", "quick")},
			{Text: "⚙️ 群组配置", CallbackData: CallbackData("admin", "config")},
		},
		{
			{Text: "🛡 封禁验证", CallbackData: CallbackData("admin", "moderation")},
			{Text: "🔍 关键词", CallbackData: CallbackData("admin", "keywords")},
		},
		{
			{Text: "🏅 等级规则", CallbackData: CallbackData("admin", "levels")},
			{Text: "🔗 邀请链接", CallbackData: CallbackData("admin", "invites")},
		},
		{
			{Text: "📋 私聊工作台", CallbackData: CallbackData("private", "home")},
			{Text: "🔙 返回首页", CallbackData: CallbackData("menu", "home")},
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
