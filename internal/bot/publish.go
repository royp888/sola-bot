package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func (a *App) handlePublish(b *gotgbot.Bot, ctx *ext.Context) error {
	content := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	if content != "" {
		return a.quickPublish(b, ctx, content)
	}
	return a.showPublishMenu(b, ctx)
}

func (a *App) handlePosts(b *gotgbot.Bot, ctx *ext.Context) error {
	if a.services.Publish == nil {
		return sendText(b, ctx, "发布服务尚未接入。", nil)
	}
	return a.showPostList(b, ctx)
}
func (a *App) routePublishCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "quick":
		return a.showPublishMenu(b, ctx)
	case "queue":
		return a.showPublishQueue(b, ctx)
	case "sync":
		return a.syncChannel(b, ctx)
	case "list":
		return a.showPostList(b, ctx)
	case "create":
		return a.startCreatePostWizard(b, ctx)
	case "private_create":
		return a.startCreatePostWizardForChat(b, ctx, payload.Resource)
	case "toggle":
		return a.togglePostFromCallback(b, ctx, payload.Resource)
	case "private_toggle":
		if len(payload.Arguments) == 0 {
			return answerCallback(b, ctx, "任务 ID 缺失")
		}
		return a.togglePostForChatFromCallback(b, ctx, payload.Resource, payload.Arguments[0])
	case "delete_confirm":
		return a.confirmDeletePostFromCallback(b, ctx, payload.Resource)
	case "private_delete_confirm":
		if len(payload.Arguments) == 0 {
			return answerCallback(b, ctx, "任务 ID 缺失")
		}
		return a.confirmDeletePostForChatFromCallback(b, ctx, payload.Resource, payload.Arguments[0])
	case "delete":
		return a.deletePostFromCallback(b, ctx, payload.Resource)
	case "private_delete":
		if len(payload.Arguments) == 0 {
			return answerCallback(b, ctx, "任务 ID 缺失")
		}
		return a.deletePostForChatFromCallback(b, ctx, payload.Resource, payload.Arguments[0])
	case "buttons":
		return respondText(b, ctx, "自动按钮可通过消息模板复用。\n\n/templates 查看模板\n/add_template 标题 | 内容 添加模板\n/del_template 模板ID 删除模板", publishMenuMarkup())
	case "once":
		return a.startCreatePostWizardWithMode(b, ctx, "once")
	case "repeat":
		return a.startCreatePostWizardWithMode(b, ctx, "repeat")
	default:
		return a.showPublishMenu(b, ctx)
	}
}

func (a *App) showPublishMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	text := strings.Join([]string{
		"📣 发布中心",
		"━━━━━━━━━━",
		"这里负责群公告、定时提醒和循环发帖。",
		"一次提醒适合活动预告；循环发布适合每日、每小时或固定间隔任务。",
		"即时发消息也可直接输入 /publish 你的内容。",
	}, "\n")
	return respondText(b, ctx, text, publishMenuMarkup())
}

func (a *App) quickPublish(b *gotgbot.Bot, ctx *ext.Context, content string) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if _, err := b.SendMessageWithContext(scope.Context, scope.Chat.ID, content, &gotgbot.SendMessageOpts{ParseMode: "HTML"}); err != nil {
		return err
	}
	if a.services.Publish != nil {
		if err := a.services.Publish.RecordQuickPost(scope.Context, scope.Chat.ID, content, scope.Actor.ID); err != nil {
			return err
		}
	}
	return respondText(b, ctx, "已发布。", publishMenuMarkup())
}

func (a *App) showPublishQueue(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.showPostList(b, ctx)
}

func (a *App) showPostList(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Publish == nil {
		return sendText(b, ctx, "发布服务尚未接入。", nil)
	}
	posts, err := a.services.Publish.ListScheduledPostItems(scope.Context, scope.Chat.ID, 10)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		return respondText(b, ctx, "📭 当前还没有定时任务。\n\n你可以直接点下方“一次提醒”或“循环发布”开始创建。", postListMarkup(nil))
	}
	var builder strings.Builder
	builder.WriteString("本群定时发帖任务：")
	for _, post := range posts {
		builder.WriteString("\n")
		builder.WriteString(formatScheduledPostLine(post))
	}
	return respondText(b, ctx, builder.String(), postListMarkup(posts))
}

func (a *App) syncChannel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Publish == nil {
		return sendText(b, ctx, "发布服务尚未接入。", nil)
	}
	if err := a.services.Publish.SyncChannel(scope.Context, scope.Chat.ID, scope.Actor.ID); err != nil {
		return err
	}
	return respondText(b, ctx, "频道同步任务已提交。", publishMenuMarkup())
}

func publishMenuMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "📋 任务列表", CallbackData: CallbackData("publish", "list")},
			{Text: "🔔 一次提醒", CallbackData: CallbackData("publish", "once")},
		},
		{
			{Text: "🔄 循环发布", CallbackData: CallbackData("publish", "repeat")},
			{Text: "🔘 自动按钮", CallbackData: CallbackData("publish", "buttons")},
		},
		{
			{Text: "🔄 频道同步", CallbackData: CallbackData("publish", "sync")},
			{Text: "🔙 返回群组", CallbackData: CallbackData("menu", "groups")},
		},
	}}}
}

func (a *App) handlePostCreate(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.startCreatePostWizard(b, ctx)
}

func (a *App) handlePostToggle(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Publish == nil {
		return sendText(b, ctx, "发布服务尚未接入。", nil)
	}
	postID, err := parsePostIDArg(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：/post_toggle 任务ID", nil)
	}
	post, err := a.services.Publish.ToggleScheduledPost(scope.Context, scope.Chat.ID, postID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, "已切换任务状态：\n"+formatScheduledPostLine(post), postListMarkup([]ScheduledPostItem{post}))
}

func (a *App) handlePostDelete(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Publish == nil {
		return sendText(b, ctx, "发布服务尚未接入。", nil)
	}
	postID, err := parsePostIDArg(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：/post_delete 任务ID", nil)
	}
	if err := a.services.Publish.DeleteScheduledPost(scope.Context, scope.Chat.ID, postID); err != nil {
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已删除定时任务 #%d。", postID), postListMarkup(nil))
}

func (a *App) togglePostFromCallback(b *gotgbot.Bot, ctx *ext.Context, rawID string) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	postID, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64)
	if err != nil || postID == 0 {
		return answerCallback(b, ctx, "任务 ID 无效")
	}
	post, err := a.services.Publish.ToggleScheduledPost(scope.Context, scope.Chat.ID, postID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, "已切换任务状态：\n"+formatScheduledPostLine(post), postListMarkup([]ScheduledPostItem{post}))
}

func (a *App) confirmDeletePostFromCallback(b *gotgbot.Bot, ctx *ext.Context, rawID string) error {
	if _, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64); err != nil {
		return answerCallback(b, ctx, "任务 ID 无效")
	}
	return respondText(b, ctx, "❓ 确认删除定时任务 #"+strings.TrimSpace(rawID)+"？", &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "✅ 确认删除", CallbackData: CallbackData("publish", "delete", strings.TrimSpace(rawID))},
				{Text: "🔙 返回列表", CallbackData: CallbackData("publish", "list")},
			},
		}},
	})
}

func (a *App) deletePostFromCallback(b *gotgbot.Bot, ctx *ext.Context, rawID string) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	postID, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64)
	if err != nil || postID == 0 {
		return answerCallback(b, ctx, "任务 ID 无效")
	}
	if err := a.services.Publish.DeleteScheduledPost(scope.Context, scope.Chat.ID, postID); err != nil {
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已删除定时任务 #%d。", postID), postListMarkup(nil))
}
func (a *App) startCreatePostWizardForChat(b *gotgbot.Bot, ctx *ext.Context, rawChatID string) error {
	chatID, err := strconv.ParseInt(strings.TrimSpace(rawChatID), 10, 64)
	if err != nil || chatID == 0 {
		return answerCallback(b, ctx, "目标群组无效")
	}
	return a.startCreatePostWizardWithTarget(b, ctx, "", chatID)
}

func (a *App) togglePostForChatFromCallback(b *gotgbot.Bot, ctx *ext.Context, rawChatID string, rawID string) error {
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	chatID, err := strconv.ParseInt(strings.TrimSpace(rawChatID), 10, 64)
	if err != nil || chatID == 0 {
		return answerCallback(b, ctx, "目标群组无效")
	}
	postID, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64)
	if err != nil || postID == 0 {
		return answerCallback(b, ctx, "任务 ID 无效")
	}
	post, err := a.services.Publish.ToggleScheduledPost(requestScope(ctx).Context, chatID, postID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, "已切换任务状态：\n"+formatScheduledPostLine(post), privatePostsMarkup(chatID, []ScheduledPostItem{post}))
}

func (a *App) confirmDeletePostForChatFromCallback(b *gotgbot.Bot, ctx *ext.Context, rawChatID string, rawID string) error {
	chatID, err := strconv.ParseInt(strings.TrimSpace(rawChatID), 10, 64)
	if err != nil || chatID == 0 {
		return answerCallback(b, ctx, "目标群组无效")
	}
	if _, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64); err != nil {
		return answerCallback(b, ctx, "任务 ID 无效")
	}
	return respondText(b, ctx, "❓ 确认删除定时任务 #"+strings.TrimSpace(rawID)+"？", &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "✅ 确认删除", CallbackData: CallbackData("publish", "private_delete", strconv.FormatInt(chatID, 10), strings.TrimSpace(rawID))},
				{Text: "🔙 返回列表", CallbackData: CallbackData("private", "posts")},
			},
		}},
	})
}

func (a *App) deletePostForChatFromCallback(b *gotgbot.Bot, ctx *ext.Context, rawChatID string, rawID string) error {
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	chatID, err := strconv.ParseInt(strings.TrimSpace(rawChatID), 10, 64)
	if err != nil || chatID == 0 {
		return answerCallback(b, ctx, "目标群组无效")
	}
	postID, err := strconv.ParseUint(strings.TrimSpace(rawID), 10, 64)
	if err != nil || postID == 0 {
		return answerCallback(b, ctx, "任务 ID 无效")
	}
	if err := a.services.Publish.DeleteScheduledPost(requestScope(ctx).Context, chatID, postID); err != nil {
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已删除定时任务 #%d。", postID), privatePostsMarkup(chatID, nil))
}

func parsePostIDArg(ctx *ext.Context) (uint64, error) {
	args := commandArgs(ctx)
	if len(args) == 0 {
		return 0, fmt.Errorf("missing post id")
	}
	return strconv.ParseUint(strings.TrimSpace(args[0]), 10, 64)
}

func formatScheduledPostLine(post ScheduledPostItem) string {
	status := "关闭"
	if post.Enabled {
		status = "启用"
	}
	schedule := "未设置"
	if strings.TrimSpace(post.CronExpr) != "" {
		schedule = strings.TrimSpace(post.CronExpr)
	} else if post.RunOnceAt != nil {
		schedule = post.RunOnceAt.Format("2006-01-02 15:04")
	}
	title := strings.TrimSpace(post.Title)
	if title == "" {
		title = strings.TrimSpace(post.Content)
	}
	if title == "" {
		title = strings.TrimSpace(post.MediaURL)
	}
	return fmt.Sprintf("#%d [%s] %s - %s", post.ID, status, schedule, truncateRunes(title, 36))
}

func postListMarkup(posts []ScheduledPostItem) *gotgbot.SendMessageOpts {
	rows := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "➕ 新建任务", CallbackData: CallbackData("publish", "create")},
			{Text: "🔄 刷新", CallbackData: CallbackData("publish", "list")},
		},
	}
	for _, post := range posts {
		id := strconv.FormatUint(post.ID, 10)
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("🔧 #%d 开关", post.ID), CallbackData: CallbackData("publish", "toggle", id)},
			{Text: "🗑 删除", CallbackData: CallbackData("publish", "delete_confirm", id)},
		})
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "🔙 返回发布", CallbackData: CallbackData("publish", "quick")}})
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}}
}
