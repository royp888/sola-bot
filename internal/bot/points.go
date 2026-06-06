package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func (a *App) handlePointsRank(b *gotgbot.Bot, ctx *ext.Context) error {
	period := "all"
	if args := commandArgs(ctx); len(args) > 0 {
		period = args[0]
	}
	return a.showPointsRankPeriod(b, ctx, period)
}

func (a *App) handlePoints(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "积分服务尚未接入。", nil)
	}
	if len(commandArgs(ctx)) > 0 {
		return a.adjustPointsByCommand(b, ctx)
	}
	summary, err := a.services.Points.GetSummary(scope.Context, scope.Chat.ID, scope.Actor.ID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, summary, pointsMenuMarkup())
}

func (a *App) handleTodayStats(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.showActivityStats(b, ctx, "today")
}

func (a *App) handleWeekStats(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.showActivityStats(b, ctx, "week")
}

func (a *App) handleCustomStats(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.showActivityStats(b, ctx, "custom")
}

func (a *App) routePointsCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "menu":
		return a.showPointsMenu(b, ctx)
	case "rank":
		period := "all"
		if payload.Resource != "" {
			period = payload.Resource
		}
		return a.showPointsRankPeriod(b, ctx, period)
	case "private_rank":
		period := "all"
		if len(payload.Arguments) > 0 && strings.TrimSpace(payload.Arguments[0]) != "" {
			period = payload.Arguments[0]
		}
		return a.showPointsRankForChatFromPrivate(b, ctx, payload.Resource, period)
	case "private_config":
		return a.showPointsConfigForChatFromPrivate(b, ctx, payload.Resource)
	case "private_stats":
		window := "today"
		if len(payload.Arguments) > 0 && strings.TrimSpace(payload.Arguments[0]) != "" {
			window = payload.Arguments[0]
		}
		return a.showActivityStatsForChatFromPrivate(b, ctx, payload.Resource, window)
	case "config":
		return a.showPointsConfigFromCallback(b, ctx)
	case "set":
		return a.setPointsConfigFromCallback(b, ctx, payload)
	case "cooldown":
		return a.setPointsCooldownFromCallback(b, ctx, payload)
	case "toggle":
		return a.togglePointsFromCallback(b, ctx)
	case "stats":
		return a.showActivityStats(b, ctx, "today")
	case "stats_week":
		return a.showActivityStats(b, ctx, "week")
	default:
		return a.showPointsMenu(b, ctx)
	}
}

func (a *App) showPointsMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	text := strings.Join([]string{
		"💎 积分中心",
		"━━━━━━━━━━",
		"这里可以查看个人积分、切换排行榜、查看活跃统计，也能进入管理员配置。",
		"管理员如需手动调分，直接回复成员消息发送 /points 10 或 /points -10 即可。",
	}, "\n")
	return respondText(b, ctx, text, pointsMenuMarkup())
}

func (a *App) showPointsRank(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.showPointsRankPeriod(b, ctx, "all")
}

func (a *App) showPointsRankPeriod(b *gotgbot.Bot, ctx *ext.Context, period string) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "积分排行服务尚未接入。", nil)
	}
	rank, err := a.services.Points.GetRank(scope.Context, scope.Chat.ID, period, 10)
	if err != nil {
		return err
	}
	return respondText(b, ctx, rank, pointsRankMarkup())
}

func (a *App) showPointsRankForChatFromPrivate(b *gotgbot.Bot, ctx *ext.Context, rawChatID string, period string) error {
	if a.services.Points == nil {
		return respondText(b, ctx, "积分排行服务尚未接入。", backPrivateHomeMarkup())
	}
	chatID, err := strconv.ParseInt(strings.TrimSpace(rawChatID), 10, 64)
	if err != nil || chatID == 0 {
		return answerCallback(b, ctx, "目标群组无效")
	}
	rank, err := a.services.Points.GetRank(requestScope(ctx).Context, chatID, period, 10)
	if err != nil {
		return err
	}
	return respondText(b, ctx, rank, privatePointsMarkup(chatID))
}

func (a *App) showPointsConfigForChatFromPrivate(b *gotgbot.Bot, ctx *ext.Context, rawChatID string) error {
	if a.services.Points == nil {
		return respondText(b, ctx, "积分配置服务尚未接入。", backPrivateHomeMarkup())
	}
	chatID, err := strconv.ParseInt(strings.TrimSpace(rawChatID), 10, 64)
	if err != nil || chatID == 0 {
		return answerCallback(b, ctx, "目标群组无效")
	}
	cfg, err := a.services.Points.GetConfig(requestScope(ctx).Context, chatID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, formatPointConfig(cfg), privatePointsMarkup(chatID))
}

func (a *App) showActivityStatsForChatFromPrivate(b *gotgbot.Bot, ctx *ext.Context, rawChatID string, window string) error {
	if a.services.Points == nil {
		return respondText(b, ctx, "活跃统计服务尚未接入。", backPrivateHomeMarkup())
	}
	chatID, err := strconv.ParseInt(strings.TrimSpace(rawChatID), 10, 64)
	if err != nil || chatID == 0 {
		return answerCallback(b, ctx, "目标群组无效")
	}
	stats, err := a.services.Points.GetActivityStats(requestScope(ctx).Context, chatID, window)
	if err != nil {
		return err
	}
	return respondText(b, ctx, stats, privatePointsMarkup(chatID))
}

func (a *App) showPointsConfigFromCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return respondText(b, ctx, "积分配置服务尚未接入。", pointsMenuMarkup())
	}
	cfg, err := a.services.Points.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, formatPointConfig(cfg), pointsConfigMarkup(cfg))
}

func (a *App) togglePointsFromCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	scope := requestScope(ctx)
	cfg, err := a.services.Points.ToggleConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	state := "已关闭"
	if cfg.Enabled {
		state = "已开启"
	}
	return respondText(b, ctx, fmt.Sprintf("积分系统%s。\n%s", state, formatPointConfig(cfg)), pointsConfigMarkup(cfg))
}

func (a *App) setPointsConfigFromCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if payload.Resource == "" || len(payload.Arguments) < 1 {
		return answerCallback(b, ctx, "配置参数缺失")
	}
	value, err := strconv.Atoi(payload.Arguments[0])
	if err != nil || value < 0 {
		return answerCallback(b, ctx, "分值无效")
	}
	scope := requestScope(ctx)
	cfg, err := a.updatePointField(scope.Context, scope.Chat.ID, payload.Resource, value)
	if err != nil {
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已更新 %s = %d\n%s", payload.Resource, value, formatPointConfig(cfg)), pointsConfigMarkup(cfg))
}

func (a *App) setPointsCooldownFromCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if payload.Resource == "" {
		return answerCallback(b, ctx, "冷却时间缺失")
	}
	seconds, err := strconv.Atoi(payload.Resource)
	if err != nil || seconds < 0 {
		return answerCallback(b, ctx, "冷却时间无效")
	}
	scope := requestScope(ctx)
	cfg, err := a.services.Points.UpdateConfig(scope.Context, scope.Chat.ID, ChatPointConfigPatch{CooldownSeconds: &seconds})
	if err != nil {
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已更新冷却时间 = %d 秒\n%s", seconds, formatPointConfig(cfg)), pointsConfigMarkup(cfg))
}

func pointsMenuMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "🏆 总榜", CallbackData: CallbackData("points", "rank", "all")},
			{Text: "📈 今日榜", CallbackData: CallbackData("points", "rank", "day")},
		},
		{
			{Text: "📊 本周榜", CallbackData: CallbackData("points", "rank", "week")},
			{Text: "📅 本月榜", CallbackData: CallbackData("points", "rank", "month")},
		},
		{
			{Text: "📈 今日统计", CallbackData: CallbackData("points", "stats")},
			{Text: "📊 本周统计", CallbackData: CallbackData("points", "stats_week")},
		},
		{
			{Text: "⚙️ 积分配置", CallbackData: CallbackData("points", "config")},
			{Text: "🔙 返回群组", CallbackData: CallbackData("menu", "groups")},
		},
	}}}
}

func pointsRankMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "🏆 总榜", CallbackData: CallbackData("points", "rank", "all")},
			{Text: "📈 今日", CallbackData: CallbackData("points", "rank", "day")},
			{Text: "📊 本周", CallbackData: CallbackData("points", "rank", "week")},
			{Text: "📅 本月", CallbackData: CallbackData("points", "rank", "month")},
		},
		{
			{Text: "🔙 返回积分中心", CallbackData: CallbackData("points", "menu")},
		},
	}}}
}

func pointsConfigMarkup(cfg ChatPointConfig) *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "🔧 开关积分", CallbackData: CallbackData("points", "toggle")},
			{Text: "🔙 返回积分中心", CallbackData: CallbackData("points", "menu")},
		},
		pointValueButtons("text", cfg.PointText),
		pointValueButtons("photo", cfg.PointPhoto),
		pointValueButtons("sticker", cfg.PointSticker),
		pointValueButtons("video", cfg.PointVideo),
		pointValueButtons("file", cfg.PointFile),
		pointValueButtons("voice", cfg.PointVoice),
		{
			{Text: "冷却 0s", CallbackData: CallbackData("points", "cooldown", "0")},
			{Text: "冷却 30s", CallbackData: CallbackData("points", "cooldown", "30")},
			{Text: "冷却 60s", CallbackData: CallbackData("points", "cooldown", "60")},
		},
	}}}
}

func pointValueButtons(field string, current int) []gotgbot.InlineKeyboardButton {
	down := current - 1
	if down < 0 {
		down = 0
	}
	up := current + 1
	return []gotgbot.InlineKeyboardButton{
		{Text: fmt.Sprintf("%s -", field), CallbackData: CallbackData("points", "set", field, strconv.Itoa(down))},
		{Text: fmt.Sprintf("%s=%d", field, current), CallbackData: CallbackData("points", "config")},
		{Text: fmt.Sprintf("%s +", field), CallbackData: CallbackData("points", "set", field, strconv.Itoa(up))},
	}
}

func pointsStatsMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "📈 今日统计", CallbackData: CallbackData("points", "stats")},
			{Text: "📊 本周统计", CallbackData: CallbackData("points", "stats_week")},
		},
		{
			{Text: "🔙 返回积分中心", CallbackData: CallbackData("points", "menu")},
		},
	}}}
}

func (a *App) adjustPointsByCommand(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}

	targetID, delta, err := pointAdjustmentTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复用户 /points 10，回复用户 /points -10，或 /points 用户ID 10。", nil)
	}
	if delta == 0 {
		return sendText(b, ctx, "调整分值不能是 0。", nil)
	}
	reason := fmt.Sprintf("bot command by %d", scope.Actor.ID)
	if err := a.services.Points.Adjust(scope.Context, scope.Chat.ID, targetID, delta, reason); err != nil {
		return err
	}
	summary, err := a.services.Points.GetSummary(scope.Context, scope.Chat.ID, targetID)
	if err != nil {
		return err
	}
	action := "增加"
	if delta < 0 {
		action = "扣除"
	}
	return respondText(b, ctx, fmt.Sprintf("已为用户 %d %s %d 积分。\n%s", targetID, action, absInt(delta), summary), pointsMenuMarkup())
}

func pointAdjustmentTarget(ctx *ext.Context) (int64, int, error) {
	args := commandArgs(ctx)
	if len(args) == 0 {
		return 0, 0, fmt.Errorf("missing args")
	}
	if ctx != nil && ctx.Message != nil {
		if ctx.Message.ReplyToMessage != nil && ctx.Message.ReplyToMessage.From != nil {
			delta, err := strconv.Atoi(args[0])
			if err != nil {
				return 0, 0, err
			}
			return ctx.Message.ReplyToMessage.From.Id, delta, nil
		}
		for _, entity := range ctx.Message.GetEntities() {
			if entity.Type == "text_mention" && entity.User != nil && len(args) >= 2 {
				delta, err := strconv.Atoi(args[len(args)-1])
				if err != nil {
					return 0, 0, err
				}
				return entity.User.Id, delta, nil
			}
		}
	}
	if len(args) < 2 {
		return 0, 0, fmt.Errorf("missing target")
	}
	if strings.HasPrefix(args[0], "@") {
		return 0, 0, fmt.Errorf("username lookup is not available")
	}
	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || targetID == 0 {
		return 0, 0, fmt.Errorf("invalid target")
	}
	delta, err := strconv.Atoi(args[1])
	if err != nil {
		return 0, 0, err
	}
	return targetID, delta, nil
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (a *App) showActivityStats(b *gotgbot.Bot, ctx *ext.Context, window string) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "活跃统计服务尚未接入。", nil)
	}
	stats, err := a.services.Points.GetActivityStats(scope.Context, scope.Chat.ID, window)
	if err != nil {
		return err
	}
	return respondText(b, ctx, stats, pointsStatsMarkup())
}
