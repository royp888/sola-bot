package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/model"
)

func (a *App) routePrivateCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "home":
		return a.showPrivateHome(b, ctx)
	case "list":
		return a.showPrivateChatList(b, ctx)
	case "select":
		chatID, err := strconv.ParseInt(payload.Resource, 10, 64)
		if err != nil || chatID == 0 {
			return answerCallback(b, ctx, "目标聊天无效")
		}
		scope := requestScope(ctx)
		if err := a.setSelectedChatID(scope.Context, scope.Actor.ID, chatID); err != nil {
			return err
		}
		return a.showPrivateConsole(b, ctx)
	case "console":
		return a.showPrivateConsole(b, ctx)
	case "posts":
		chat, ok, err := a.selectedPrivateChat(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return a.showPrivateChatList(b, ctx)
		}
		return a.showScheduledPostsForChat(b, ctx, chat.ChatID, privateConsoleMarkup(chat))
	case "points":
		chat, ok, err := a.selectedPrivateChat(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return a.showPrivateChatList(b, ctx)
		}
		return a.showPointsMenuForChat(b, ctx, chat.ChatID, privateConsoleMarkup(chat))
	case "admin":
		chat, ok, err := a.selectedPrivateChat(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return a.showPrivateChatList(b, ctx)
		}
		return a.showPrivateAdminCenter(b, ctx, chat)
	case "summary":
		chat, ok, err := a.selectedPrivateChat(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return a.showPrivateChatList(b, ctx)
		}
		return a.showPrivateSummary(b, ctx, chat)
	case "lottery_create":
		chat, ok, err := a.selectedPrivateChat(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return a.showPrivateChatList(b, ctx)
		}
		if !isGroupChatType(chat.ChatType) {
			return respondText(b, ctx, "抽奖只能创建到群组或超级群。", privateConsoleMarkup(chat))
		}
		return respondText(b, ctx, "请选择抽奖类型：\n\n按钮抽奖：群成员点击公告按钮参与。\n口令抽奖：群成员发送指定口令参与。\n双模式：按钮和口令都能参与。", lotteryCreateTypePrivateMarkup())
	case "lottery_button_create", "lottery_keyword_create", "lottery_both_create":
		chat, ok, err := a.selectedPrivateChat(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return a.showPrivateChatList(b, ctx)
		}
		if !isGroupChatType(chat.ChatType) {
			return respondText(b, ctx, "抽奖只能创建到群组或超级群。", privateConsoleMarkup(chat))
		}
		joinType := "button"
		if payload.Action == "lottery_keyword_create" {
			joinType = "keyword"
		}
		if payload.Action == "lottery_both_create" {
			joinType = "both"
		}
		return a.startCreateLotteryWizardWithJoinType(b, ctx, chat.ChatID, joinType)
	case "lottery_active":
		chat, ok, err := a.selectedPrivateChat(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return a.showPrivateChatList(b, ctx)
		}
		return a.showPrivateLotteryCenter(b, ctx, chat)
	default:
		return answerCallback(b, ctx, "未知操作")
	}
}

func (a *App) showPrivateHome(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	chats, err := a.listPrivateManagedChats(ctx)
	if err != nil {
		return err
	}
	text := strings.Join([]string{
		"运营控制台",
		"",
		"在私聊里选择一个已绑定的群组或频道，就能像面板一样继续操作。",
		"支持切换目标、创建抽奖、查看积分、定时发帖和群管入口。",
	}, "\n")
	if len(chats) == 0 {
		text += "\n\n当前还没有可管理的目标，请先把 Bot 加进群组或频道，并在对应聊天里发送 /bind。"
		return respondText(b, ctx, text, privateHomeMarkup(false, false))
	}
	selectedLabel := "未选择"
	if chat, ok := a.currentPrivateChat(scope.Context, chats, scope.Actor.ID); ok {
		selectedLabel = chatTypeLabel(chat.ChatType) + " · " + chatTitle(chat)
	}
	text += fmt.Sprintf("\n\n已绑定目标：%d\n当前目标：%s", len(chats), selectedLabel)
	return respondText(b, ctx, text, privateHomeMarkup(true, selectedLabel != "未选择"))
}

func (a *App) showPrivateChatList(b *gotgbot.Bot, ctx *ext.Context) error {
	chats, err := a.listPrivateManagedChats(ctx)
	if err != nil {
		return err
	}
	if len(chats) == 0 {
		text := strings.Join([]string{
			"还没有可管理的已绑定群组或频道。",
			"",
			"请先把我加入目标群组/频道并设为管理员，然后在目标聊天里发送 /bind 或 /check_admin。",
		}, "\n")
		return respondText(b, ctx, text, nil)
	}

	rows := make([][]gotgbot.InlineKeyboardButton, 0, len(chats)+1)
	for _, chat := range chats {
		label := fmt.Sprintf("%s %s", chatTypeLabel(chat.ChatType), chatTitle(chat))
		rows = append(rows, []gotgbot.InlineKeyboardButton{{
			Text:         truncateButtonText(label, 28),
			CallbackData: CallbackData("private", "select", strconv.FormatInt(chat.ChatID, 10)),
		}})
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "返回首页", CallbackData: CallbackData("private", "home")}})
	text := "请选择当前要管理的目标。\n\n后续所有按钮都会针对你选中的群组或频道执行。"
	return respondText(b, ctx, text, &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}})
}

func (a *App) showPrivateConsole(b *gotgbot.Bot, ctx *ext.Context) error {
	chat, ok, err := a.selectedPrivateChat(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return a.showPrivateChatList(b, ctx)
	}
	text := strings.Join([]string{
		"🎛 私聊工作台",
		"━━━━━━━━━━",
		fmt.Sprintf("当前目标：%s", chatTitle(chat)),
		fmt.Sprintf("类型：%s", chatTypeLabel(chat.ChatType)),
		fmt.Sprintf("Chat ID：%d", chat.ChatID),
		"",
		"这里负责管理动作：创建抽奖、看积分、管定时任务、进群管中心。",
		"群里负责成员交互：参与抽奖、发送口令、查看活动结果。",
	}, "\n")
	return respondText(b, ctx, text, privateConsoleMarkup(chat))
}

func (a *App) listPrivateManagedChats(ctx *ext.Context) ([]api.ChatBinding, error) {
	scope := requestScope(ctx)
	if scope.Actor.ID == 0 || a.services.ChatBindings == nil {
		return []api.ChatBinding{}, nil
	}
	chats, err := a.services.ChatBindings.ListByTelegramUser(scope.Context, scope.Actor.ID, 100)
	if err != nil {
		return nil, err
	}
	filtered := make([]api.ChatBinding, 0, len(chats))
	for _, chat := range chats {
		if isManagedChatType(chat.ChatType) {
			filtered = append(filtered, chat)
		}
	}
	return filtered, nil
}

func (a *App) currentPrivateChat(ctx context.Context, chats []api.ChatBinding, userID int64) (api.ChatBinding, bool) {
	if selectedID, ok := a.getSelectedChatID(ctx, userID); ok {
		for _, chat := range chats {
			if chat.ChatID == selectedID {
				return chat, true
			}
		}
	}
	if len(chats) == 0 {
		return api.ChatBinding{}, false
	}
	return chats[0], true
}

func (a *App) selectedPrivateChat(ctx *ext.Context) (api.ChatBinding, bool, error) {
	scope := requestScope(ctx)
	chats, err := a.listPrivateManagedChats(ctx)
	if err != nil {
		return api.ChatBinding{}, false, err
	}
	if len(chats) == 0 {
		return api.ChatBinding{}, false, nil
	}
	chat, ok := a.currentPrivateChat(scope.Context, chats, scope.Actor.ID)
	if !ok {
		return api.ChatBinding{}, false, nil
	}
	if err := a.setSelectedChatID(scope.Context, scope.Actor.ID, chat.ChatID); err != nil {
		return api.ChatBinding{}, false, err
	}
	return chat, true, nil
}

func privateHomeMarkup(hasChats bool, hasSelection bool) *gotgbot.SendMessageOpts {
	rows := [][]gotgbot.InlineKeyboardButton{}
	if hasChats {
		rows = append(rows,
			[]gotgbot.InlineKeyboardButton{{Text: "选择目标", CallbackData: CallbackData("private", "list")}},
		)
		if hasSelection {
			rows = append(rows,
				[]gotgbot.InlineKeyboardButton{{Text: "进入工作台", CallbackData: CallbackData("private", "console")}},
			)
		}
	}
	rows = append(rows,
		[]gotgbot.InlineKeyboardButton{{Text: "私聊功能说明", CallbackData: CallbackData("menu", "private_help")}},
	)
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}}
}

func privateConsoleMarkup(chat api.ChatBinding) *gotgbot.SendMessageOpts {
	chatResource := strconv.FormatInt(chat.ChatID, 10)
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "抽奖中心", CallbackData: CallbackData("private", "lottery_active", chatResource)},
			{Text: "创建抽奖", CallbackData: CallbackData("private", "lottery_create", chatResource)},
		},
		{
			{Text: "积分中心", CallbackData: CallbackData("private", "points", chatResource)},
			{Text: "定时发帖", CallbackData: CallbackData("private", "posts", chatResource)},
		},
		{
			{Text: "群管中心", CallbackData: CallbackData("private", "admin", chatResource)},
			{Text: "运行概览", CallbackData: CallbackData("private", "summary", chatResource)},
		},
		{
			{Text: "切换目标", CallbackData: CallbackData("private", "list")},
			{Text: "返回首页", CallbackData: CallbackData("private", "home")},
		},
	}}}
}

func (a *App) showPrivateSummary(b *gotgbot.Bot, ctx *ext.Context, chat api.ChatBinding) error {
	scope := requestScope(ctx)
	lines := []string{
		"运行概览",
		"",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		fmt.Sprintf("类型：%s", chatTypeLabel(chat.ChatType)),
		fmt.Sprintf("Chat ID：%d", chat.ChatID),
	}
	if a.services.Points != nil {
		stats, err := a.services.Points.GetActivityStats(scope.Context, chat.ChatID, "today")
		if err == nil && strings.TrimSpace(stats) != "" {
			lines = append(lines, "", stats)
		}
	}
	return respondText(b, ctx, strings.Join(lines, "\n"), privateConsoleMarkup(chat))
}

func (a *App) showPrivateLotteryCenter(b *gotgbot.Bot, ctx *ext.Context, chat api.ChatBinding) error {
	if a.services.Lottery == nil {
		return respondText(b, ctx, "抽奖服务尚未接入。", privateConsoleMarkup(chat))
	}
	activeItems, err := a.services.Lottery.ListActiveItems(requestScope(ctx).Context, chat.ChatID, 6)
	if err != nil {
		return err
	}
	lines := []string{
		"🎁 抽奖中心",
		"━━━━━━━━━━",
		fmt.Sprintf("当前目标：%s", chatTitle(chat)),
		fmt.Sprintf("目标类型：%s", chatTypeLabel(chat.ChatType)),
		fmt.Sprintf("当前进行中：%d 场", len(activeItems)),
		"",
	}
	if len(activeItems) > 0 {
		lines = append(lines, "当前进行中的抽奖活动")
		for idx, item := range activeItems {
			lines = append(lines, fmt.Sprintf("%d. %s · %s", idx+1, lotteryTextFallback(item.Title, "未命名抽奖"), lotteryJoinTypeLabel(item.JoinType)))
			lines = append(lines, fmt.Sprintf("奖品：%s", lotteryTextFallback(item.Prize, "未填写")))
			lines = append(lines, fmt.Sprintf("参与：%d / 中奖：%d", item.EntryCount, maxInt(item.WinnerCount, 1)))
			if item.EndAt != nil {
				lines = append(lines, fmt.Sprintf("开奖时间：%s", formatChinaTime(*item.EndAt)))
			}
			lines = append(lines, "")
		}
	} else {
		lines = append(lines, "暂无抽奖", "")
	}
	lines = append(lines,
		"私聊负责创建、查看和取消活动；群里负责成员参与、口令触发和结果传播。",
	)
	return respondText(b, ctx, strings.Join(lines, "\n"), privateLotteryMarkup(chat, activeItems))
}

func privateLotteryMarkup(chat api.ChatBinding, items []api.Lottery) *gotgbot.SendMessageOpts {
	chatResource := strconv.FormatInt(chat.ChatID, 10)
	rows := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "按钮抽奖", CallbackData: CallbackData("private", "lottery_button_create", chatResource)},
			{Text: "口令抽奖", CallbackData: CallbackData("private", "lottery_keyword_create", chatResource)},
		},
		{
			{Text: "双模式", CallbackData: CallbackData("private", "lottery_both_create", chatResource)},
			{Text: "刷新中心", CallbackData: CallbackData("private", "lottery_active", chatResource)},
		},
	}
	for _, item := range items {
		id := strconv.FormatInt(item.ID, 10)
		title := lotteryTextFallback(item.Title, fmt.Sprintf("#%d", item.ID))
		if len([]rune(title)) > 10 {
			title = string([]rune(title)[:10]) + "..."
		}
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: "详情 " + title, CallbackData: CallbackData("lottery", "info", id)},
			{Text: "取消", CallbackData: CallbackData("lottery", "cancel", id)},
		})
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "返回工作台", CallbackData: CallbackData("private", "console")}})
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}}
}
func (a *App) showPrivateAdminCenter(b *gotgbot.Bot, ctx *ext.Context, chat api.ChatBinding) error {
	scope := requestScope(ctx)
	lines := []string{
		"群管中心",
		"",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"群组里可直接使用：/ban /mute /kick /warn /unwarn /warns",
		"入群验证、欢迎语、警告上限请在后台或群内命令里配置。",
	}
	if a.services.TelegramAccess != nil {
		status, err := a.services.TelegramAccess.CheckBotAdmin(scope.Context, b, chat.ChatID)
		if err == nil {
			lines = append(lines, "", fmt.Sprintf("Bot 管理状态：%s", status.Status))
		}
	}
	return respondText(b, ctx, strings.Join(lines, "\n"), privateConsoleMarkup(chat))
}

func (a *App) showPointsMenuForChat(b *gotgbot.Bot, ctx *ext.Context, chatID int64, back *gotgbot.SendMessageOpts) error {
	if a.services.Points == nil {
		return respondText(b, ctx, "积分服务尚未接入。", back)
	}
	stats, err := a.services.Points.GetActivityStats(requestScope(ctx).Context, chatID, "today")
	if err != nil {
		return err
	}
	text := strings.Join([]string{
		"积分中心",
		"",
		"自动计分、防刷冷却、排行榜和手动调分都在这里。",
		stats,
	}, "\n")
	return respondText(b, ctx, text, privatePointsMarkup(chatID))
}

func privatePointsMarkup(chatID int64) *gotgbot.SendMessageOpts {
	chatResource := strconv.FormatInt(chatID, 10)
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "总榜", CallbackData: CallbackData("points", "private_rank", chatResource, "all")},
			{Text: "今日榜", CallbackData: CallbackData("points", "private_rank", chatResource, "day")},
		},
		{
			{Text: "本周榜", CallbackData: CallbackData("points", "private_rank", chatResource, "week")},
			{Text: "积分配置", CallbackData: CallbackData("points", "private_config", chatResource)},
		},
		{
			{Text: "今日统计", CallbackData: CallbackData("points", "private_stats", chatResource, "today")},
			{Text: "本周统计", CallbackData: CallbackData("points", "private_stats", chatResource, "week")},
		},
		{
			{Text: "返回工作台", CallbackData: CallbackData("private", "console")},
		},
	}}}
}

func (a *App) showScheduledPostsForChat(b *gotgbot.Bot, ctx *ext.Context, chatID int64, back *gotgbot.SendMessageOpts) error {
	if a.services.Publish == nil {
		return respondText(b, ctx, "发布服务尚未接入。", back)
	}
	posts, err := a.services.Publish.ListScheduledPostItems(requestScope(ctx).Context, chatID, 10)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		return respondText(b, ctx, "当前目标暂无定时任务。\n\n可用命令：/post_create、/post_toggle ID、/post_delete ID", privatePostsMarkup(chatID, nil))
	}
	var builder strings.Builder
	builder.WriteString("当前目标的定时任务\n\n")
	for _, post := range posts {
		builder.WriteString(formatScheduledPostLine(post))
		builder.WriteString("\n")
	}
	builder.WriteString("\n可继续使用 /post_create 创建任务。")
	return respondText(b, ctx, strings.TrimSpace(builder.String()), privatePostsMarkup(chatID, posts))
}

func privatePostsMarkup(chatID int64, posts []ScheduledPostItem) *gotgbot.SendMessageOpts {
	chatResource := strconv.FormatInt(chatID, 10)
	rows := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "新建任务", CallbackData: CallbackData("publish", "private_create", chatResource)},
			{Text: "刷新列表", CallbackData: CallbackData("private", "posts")},
		},
	}
	for _, post := range posts {
		id := strconv.FormatUint(post.ID, 10)
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("#%d 开关", post.ID), CallbackData: CallbackData("publish", "private_toggle", chatResource, id)},
			{Text: "删除", CallbackData: CallbackData("publish", "private_delete_confirm", chatResource, id)},
		})
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "返回工作台", CallbackData: CallbackData("private", "console")}})
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}}
}

func lotteryCreateTypePrivateMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "按钮抽奖", CallbackData: CallbackData("private", "lottery_button_create")},
			{Text: "口令抽奖", CallbackData: CallbackData("private", "lottery_keyword_create")},
		},
		{
			{Text: "双模式", CallbackData: CallbackData("private", "lottery_both_create")},
			{Text: "返回工作台", CallbackData: CallbackData("private", "console")},
		},
	}}}
}

func chatTitle(chat api.ChatBinding) string {
	if strings.TrimSpace(chat.Title) != "" {
		return strings.TrimSpace(chat.Title)
	}
	if strings.TrimSpace(chat.Username) != "" {
		return "@" + strings.TrimSpace(chat.Username)
	}
	return strconv.FormatInt(chat.ChatID, 10)
}

func chatTypeLabel(chatType string) string {
	switch strings.ToLower(strings.TrimSpace(chatType)) {
	case "channel":
		return "频道"
	case "supergroup":
		return "超级群"
	case "group":
		return "群组"
	default:
		return chatType
	}
}

func isGroupChatType(chatType string) bool {
	chatType = strings.ToLower(strings.TrimSpace(chatType))
	return chatType == "group" || chatType == "supergroup"
}

func isManagedChatType(chatType string) bool {
	return isGroupChatType(chatType) || strings.EqualFold(strings.TrimSpace(chatType), "channel")
}

func truncateButtonText(text string, max int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

var _ = model.User{}
