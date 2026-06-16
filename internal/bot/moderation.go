package bot

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
)

func (a *App) registerModerationHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("ban", a.wrap(a.handleBan, a.RateLimit("cmd:ban", 1))))
	d.AddHandler(handlers.NewCommand("bans", a.wrap(a.handleBans, a.RateLimit("cmd:bans", 1))))
	d.AddHandler(handlers.NewCommand("violations", a.wrap(a.handleViolations, a.RateLimit("cmd:violations", 1))))
	d.AddHandler(handlers.NewCommand("resolve_violation", a.wrap(a.handleResolveViolation, a.RateLimit("cmd:resolve_violation", 1))))
	d.AddHandler(handlers.NewCommand("ignore_violation", a.wrap(a.handleIgnoreViolation, a.RateLimit("cmd:ignore_violation", 1))))
	d.AddHandler(handlers.NewCommand("manage", a.wrap(a.handleManageMember, a.RateLimit("cmd:manage", 1))))
	d.AddHandler(handlers.NewCommand("unban", a.wrap(a.handleUnban, a.RateLimit("cmd:unban", 1))))
	d.AddHandler(handlers.NewCommand("mute", a.wrap(a.handleMute, a.RateLimit("cmd:mute", 1))))
	d.AddHandler(handlers.NewCommand("unmute", a.wrap(a.handleUnmute, a.RateLimit("cmd:unmute", 1))))
	d.AddHandler(handlers.NewCommand("kick", a.wrap(a.handleKick, a.RateLimit("cmd:kick", 1))))
	d.AddHandler(handlers.NewCommand("warn", a.wrap(a.handleWarn, a.RateLimit("cmd:warn", 1))))
	d.AddHandler(handlers.NewCommand("unwarn", a.wrap(a.handleUnwarn, a.RateLimit("cmd:unwarn", 1))))
	d.AddHandler(handlers.NewCommand("warns", a.wrap(a.handleWarns, a.RateLimit("cmd:warns", 1))))
	d.AddHandler(handlers.NewMessage(message.All, a.handleMessageModeration))
	d.AddHandler(handlers.NewCommand("purge", a.wrap(a.handlePurge, a.RateLimit("cmd:purge", 1))))
	d.AddHandler(handlers.NewCommand("del", a.wrap(a.handleDel, a.RateLimit("cmd:del", 1))))
	d.AddHandler(handlers.NewCommand("promote", a.wrap(a.handlePromote, a.RateLimit("cmd:promote", 1))))
	d.AddHandler(handlers.NewCommand("demote", a.wrap(a.handleDemote, a.RateLimit("cmd:demote", 1))))
	d.AddHandler(handlers.NewCommand("set_title", a.wrap(a.handleSetTitle, a.RateLimit("cmd:set_title", 1))))
	d.AddHandler(handlers.NewCommand("report", a.wrap(a.handleReport, a.RateLimit("cmd:report", 3))))
	d.AddHandler(handlers.NewCommand("ban_ghosts", a.wrap(a.handleBanGhosts, a.RateLimit("cmd:ban_ghosts", 1))))
}

func (a *App) auditModAction(actorID int64, chatID int64, action string, targetID int64, detail string) {
	if a.services.AuditLog != nil {
		a.services.AuditLog.Log(AuditLogEntry{
			ActorTelegramID:  actorID,
			ChatTelegramID:   chatID,
			Action:           action,
			EntityType:       "user",
			TargetTelegramID: targetID,
			Detail:           detail,
		})
	}
}

func (a *App) handleBan(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	ids, reason, err := parseBatchModerationTargets(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标消息 /ban 原因，或 /ban <user_id1> [user_id2...] 原因。", nil)
	}
	doBan := func(targetID int64) error {
		if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		if a.services.Admin != nil {
			_ = a.services.Admin.RecordBan(scope.Context, scope.Chat.ID, targetID, scope.Actor.ID, reason)
		}
		a.auditModAction(scope.Actor.ID, scope.Chat.ID, "ban", targetID, reason)
		return nil
	}
	if len(ids) == 1 {
		if err := doBan(ids[0]); err != nil {
			return err
		}
		return sendText(b, ctx, fmt.Sprintf("已封禁用户 %d。%s", ids[0], reasonSuffix(reason)), nil)
	}
	var successes, failures []string
	for _, id := range ids {
		if err := doBan(id); err != nil {
			failures = append(failures, fmt.Sprintf("%d", id))
		} else {
			successes = append(successes, fmt.Sprintf("%d", id))
		}
	}
	return sendText(b, ctx, batchResultMessage("封禁", successes, failures), nil)
}

func (a *App) handleUnban(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.withModerationTarget(b, ctx, "解封", func(scope RequestScope, targetID int64, reason string) error {
		if _, err := b.UnbanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.UnbanChatMemberOpts{OnlyIfBanned: true}); err != nil {
			return err
		}
		if a.services.Admin != nil {
			if err := a.services.Admin.RecordUnban(scope.Context, scope.Chat.ID, targetID, scope.Actor.ID); err != nil {
				return err
			}
		}
		return sendText(b, ctx, fmt.Sprintf("已解封用户 %d。", targetID), nil)
	})
}

func (a *App) handleMute(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	ids, duration, err := parseBatchMuteTargets(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标消息 /mute 30m，或 /mute <user_id1> [user_id2...] 30m。", nil)
	}
	until := time.Now().Add(duration).Unix()
	doMute := func(targetID int64) error {
		if _, err := b.RestrictChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, mutePermissions(), &gotgbot.RestrictChatMemberOpts{UntilDate: until, UseIndependentChatPermissions: true}); err != nil {
			return err
		}
		a.auditModAction(scope.Actor.ID, scope.Chat.ID, "mute", targetID, fmt.Sprintf("duration: %s", duration.Round(time.Second)))
		return nil
	}
	if len(ids) == 1 {
		if err := doMute(ids[0]); err != nil {
			return err
		}
		return sendText(b, ctx, fmt.Sprintf("已禁言用户 %d，时长 %s。", ids[0], duration.Round(time.Second)), nil)
	}
	var successes, failures []string
	for _, id := range ids {
		if err := doMute(id); err != nil {
			failures = append(failures, fmt.Sprintf("%d", id))
		} else {
			successes = append(successes, fmt.Sprintf("%d", id))
		}
	}
	return sendText(b, ctx, batchResultMessage(fmt.Sprintf("禁言（%s）", duration.Round(time.Second)), successes, failures), nil)
}

func (a *App) handleUnmute(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, _, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标消息 /unmute，或 /unmute <user_id>。", nil)
	}
	if _, err := b.RestrictChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, fullPermissions(), &gotgbot.RestrictChatMemberOpts{UseIndependentChatPermissions: true}); err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("已解除用户 %d 的禁言。", targetID), nil)
}

func (a *App) handleKick(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	ids, reason, err := parseBatchModerationTargets(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标消息 /kick 原因，或 /kick <user_id1> [user_id2...] 原因。", nil)
	}
	doKick := func(targetID int64) error {
		if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		_, _ = b.UnbanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.UnbanChatMemberOpts{})
		a.auditModAction(scope.Actor.ID, scope.Chat.ID, "kick", targetID, reason)
		return nil
	}
	if len(ids) == 1 {
		if err := doKick(ids[0]); err != nil {
			return err
		}
		return sendText(b, ctx, fmt.Sprintf("已踢出用户 %d。%s", ids[0], reasonSuffix(reason)), nil)
	}
	var successes, failures []string
	for _, id := range ids {
		if err := doKick(id); err != nil {
			failures = append(failures, fmt.Sprintf("%d", id))
		} else {
			successes = append(successes, fmt.Sprintf("%d", id))
		}
	}
	return sendText(b, ctx, batchResultMessage("踢出", successes, failures), nil)
}

func (a *App) handleWarn(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, args, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标消息 /warn 原因，或 /warn <user_id> 原因。", nil)
	}
	reason := strings.TrimSpace(strings.Join(args, " "))
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组管理服务尚未接入。", nil)
	}
	count, limit, err := a.services.Admin.RecordWarn(scope.Context, scope.Chat.ID, targetID, scope.Actor.ID, reason)
	if err != nil {
		return err
	}
	if limit > 0 && int(count) >= limit {
		if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		_ = a.services.Admin.RecordBan(scope.Context, scope.Chat.ID, targetID, scope.Actor.ID, fmt.Sprintf("警告达到上限 %d", limit))
		return sendText(b, ctx, fmt.Sprintf("用户 %d 已被警告 %d/%d，达到上限，已自动封禁。", targetID, count, limit), nil)
	}
	return sendText(b, ctx, fmt.Sprintf("已警告用户 %d，当前 %d/%d。%s", targetID, count, limit, reasonSuffix(reason)), nil)
}

func (a *App) handleUnwarn(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, _, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标消息 /unwarn，或 /unwarn <user_id>。", nil)
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组管理服务尚未接入。", nil)
	}
	if err := a.services.Admin.ClearWarns(scope.Context, scope.Chat.ID, targetID); err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("已清除用户 %d 的未处理警告。", targetID), nil)
}

func (a *App) handleWarns(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, _, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标消息 /warns，或 /warns <user_id>。", nil)
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组管理服务尚未接入。", nil)
	}
	count, err := a.services.Admin.CountWarns(scope.Context, scope.Chat.ID, targetID)
	if err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("用户 %d 当前未清除警告：%d。", targetID, count), nil)
}

func (a *App) handleManageMember(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, _, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "请回复目标用户消息发送 /manage，或使用 /manage <user_id>。", nil)
	}
	if targetID == scope.Actor.ID {
		return sendText(b, ctx, "不能对自己执行管理操作。", nil)
	}
	text := fmt.Sprintf("👤 成员管理\n目标用户：%d\n\n请选择要执行的操作。禁言和警告会立即执行；封禁和踢出请二次确认。", targetID)
	return sendText(b, ctx, text, moderationTargetMarkup(targetID, false))
}

func (a *App) showModerationMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	text := strings.Join([]string{
		"🛡 群管中心",
		"━━━━━━━━━━",
		"回复成员消息发送 /manage，可以打开这个成员的快捷管理面板。",
		"常用动作包括禁言 30 分钟、禁言 2 小时、警告、踢出、封禁、查看警告次数。",
		"需要全局记录时，可继续查看违规记录和封禁列表。",
	}, "\n")
	return respondText(b, ctx, text, moderationMenuMarkup())
}

func (a *App) handleModerationCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, err := strconv.ParseInt(payload.Resource, 10, 64)
	if err != nil || targetID == 0 {
		return answerCallback(b, ctx, "目标用户无效")
	}
	action := ""
	if len(payload.Arguments) > 0 {
		action = payload.Arguments[0]
	}
	scope := requestScope(ctx)
	switch action {
	case "panel":
		return respondText(b, ctx, fmt.Sprintf("成员管理\n目标用户：%d\n\n请选择操作。", targetID), moderationTargetMarkup(targetID, false))
	case "mute30":
		return a.restrictFromPanel(b, ctx, scope, targetID, 30*time.Minute)
	case "mute2h":
		return a.restrictFromPanel(b, ctx, scope, targetID, 2*time.Hour)
	case "unmute":
		if _, err := b.RestrictChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, fullPermissions(), &gotgbot.RestrictChatMemberOpts{UseIndependentChatPermissions: true}); err != nil {
			return err
		}
		return respondText(b, ctx, fmt.Sprintf("已解除用户 %d 的禁言。", targetID), moderationTargetMarkup(targetID, false))
	case "warn":
		if a.services.Admin == nil {
			return respondText(b, ctx, "群组管理服务尚未接入。", moderationTargetMarkup(targetID, false))
		}
		count, limit, err := a.services.Admin.RecordWarn(scope.Context, scope.Chat.ID, targetID, scope.Actor.ID, "button warning")
		if err != nil {
			return err
		}
		if limit > 0 && int(count) >= limit {
			if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
				return err
			}
			_ = a.services.Admin.RecordBan(scope.Context, scope.Chat.ID, targetID, scope.Actor.ID, fmt.Sprintf("警告达到上限 %d", limit))
			return respondText(b, ctx, fmt.Sprintf("用户 %d 已被警告 %d/%d，达到上限，已自动封禁。", targetID, count, limit), moderationTargetMarkup(targetID, false))
		}
		return respondText(b, ctx, fmt.Sprintf("已警告用户 %d，当前 %d/%d。", targetID, count, limit), moderationTargetMarkup(targetID, false))
	case "warns":
		if a.services.Admin == nil {
			return respondText(b, ctx, "群组管理服务尚未接入。", moderationTargetMarkup(targetID, false))
		}
		count, err := a.services.Admin.CountWarns(scope.Context, scope.Chat.ID, targetID)
		if err != nil {
			return err
		}
		return respondText(b, ctx, fmt.Sprintf("用户 %d 当前未清除警告：%d。", targetID, count), moderationTargetMarkup(targetID, false))
	case "ban_confirm":
		return respondText(b, ctx, fmt.Sprintf("确认封禁用户 %d？", targetID), moderationConfirmMarkup(targetID, "ban"))
	case "kick_confirm":
		return respondText(b, ctx, fmt.Sprintf("确认踢出用户 %d？", targetID), moderationConfirmMarkup(targetID, "kick"))
	case "ban":
		if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		if a.services.Admin != nil {
			_ = a.services.Admin.RecordBan(scope.Context, scope.Chat.ID, targetID, scope.Actor.ID, "button ban")
		}
		return respondText(b, ctx, fmt.Sprintf("已封禁用户 %d。", targetID), moderationMenuMarkup())
	case "kick":
		if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		if _, err := b.UnbanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.UnbanChatMemberOpts{}); err != nil {
			return err
		}
		return respondText(b, ctx, fmt.Sprintf("已踢出用户 %d。", targetID), moderationMenuMarkup())
	default:
		return respondText(b, ctx, "未知成员管理操作。", moderationTargetMarkup(targetID, false))
	}
}

func (a *App) restrictFromPanel(b *gotgbot.Bot, ctx *ext.Context, scope RequestScope, targetID int64, duration time.Duration) error {
	until := time.Now().Add(duration).Unix()
	if _, err := b.RestrictChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, mutePermissions(), &gotgbot.RestrictChatMemberOpts{UntilDate: until, UseIndependentChatPermissions: true}); err != nil {
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已禁言用户 %d，时长 %s。", targetID, duration.Round(time.Second)), moderationTargetMarkup(targetID, false))
}

func moderationMenuMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "📋 成员管理说明", CallbackData: CallbackData("admin", "moderation")},
			{Text: "✅ 入群验证", CallbackData: CallbackData("admin", "verify_menu")},
		},
		{
			{Text: "🔍 关键词过滤", CallbackData: CallbackData("admin", "keywords")},
			{Text: "📝 消息模板", CallbackData: CallbackData("admin", "templates")},
		},
		{
			{Text: "⚠️ 违规记录", CallbackData: CallbackData("admin", "violations")},
			{Text: "🚫 封禁记录", CallbackData: CallbackData("admin", "bans")},
		},
		{
			{Text: "🏅 等级规则", CallbackData: CallbackData("admin", "levels")},
			{Text: "🔗 邀请链接", CallbackData: CallbackData("admin", "invites")},
		},
		{
			{Text: "🔙 返回群组", CallbackData: CallbackData("menu", "groups")},
		},
	}}}
}

func moderationTargetMarkup(targetID int64, compact bool) *gotgbot.SendMessageOpts {
	id := strconv.FormatInt(targetID, 10)
	rows := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "🔇 禁言 30m", CallbackData: CallbackData("admin", "mod", id, "mute30")},
			{Text: "🔇 禁言 2h", CallbackData: CallbackData("admin", "mod", id, "mute2h")},
			{Text: "🔊 解除禁言", CallbackData: CallbackData("admin", "mod", id, "unmute")},
		},
		{
			{Text: "⚠️ 警告", CallbackData: CallbackData("admin", "mod", id, "warn")},
			{Text: "📋 查看警告", CallbackData: CallbackData("admin", "mod", id, "warns")},
		},
		{
			{Text: "🚫 封禁", CallbackData: CallbackData("admin", "mod", id, "ban_confirm")},
			{Text: "👢 踢出", CallbackData: CallbackData("admin", "mod", id, "kick_confirm")},
		},
	}
	if !compact {
		rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "🔙 返回群组", CallbackData: CallbackData("menu", "groups")}})
	}
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}}
}

func moderationAssetsMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "🔍 关键词过滤", CallbackData: CallbackData("admin", "keywords")},
			{Text: "📝 消息模板", CallbackData: CallbackData("admin", "templates")},
		},
		{
			{Text: "⚠️ 违规记录", CallbackData: CallbackData("admin", "violations")},
			{Text: "🏅 等级规则", CallbackData: CallbackData("admin", "levels")},
		},
		{
			{Text: "🔗 邀请链接", CallbackData: CallbackData("admin", "invites")},
			{Text: "🔙 返回群管", CallbackData: CallbackData("admin", "moderation")},
		},
	}}}
}

func formatAdminFeaturePanel(title string, body string, actions ...string) string {
	lines := []string{title, "━━━━━━━━━━", body}
	if len(actions) > 0 {
		lines = append(lines, "", "常用命令")
		lines = append(lines, actions...)
	}
	return strings.Join(lines, "\n")
}

func moderationConfirmMarkup(targetID int64, action string) *gotgbot.SendMessageOpts {
	id := strconv.FormatInt(targetID, 10)
	label := "✅ 确认"
	if action == "ban" {
		label = "✅ 确认封禁"
	}
	if action == "kick" {
		label = "✅ 确认踢出"
	}
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: label, CallbackData: CallbackData("admin", "mod", id, action)},
			{Text: "🔙 返回", CallbackData: CallbackData("admin", "mod", id, "panel")},
		},
	}}}
}

func parseBatchModerationTargets(ctx *ext.Context) ([]int64, string, error) {
	if ctx != nil && ctx.Message != nil && ctx.Message.ReplyToMessage != nil && ctx.Message.ReplyToMessage.From != nil {
		args := commandArgs(ctx)
		return []int64{ctx.Message.ReplyToMessage.From.Id}, strings.TrimSpace(strings.Join(args, " ")), nil
	}
	args := commandArgs(ctx)
	if len(args) == 0 {
		return nil, "", fmt.Errorf("missing targets")
	}
	if strings.HasPrefix(args[0], "@") {
		return nil, "", fmt.Errorf("username lookup not supported")
	}
	var ids []int64
	i := 0
	for i < len(args) {
		id, err := strconv.ParseInt(args[i], 10, 64)
		if err != nil || id == 0 {
			break
		}
		ids = append(ids, id)
		i++
	}
	if len(ids) == 0 {
		return nil, "", fmt.Errorf("no valid user IDs")
	}
	return ids, strings.TrimSpace(strings.Join(args[i:], " ")), nil
}

func parseBatchMuteTargets(ctx *ext.Context) ([]int64, time.Duration, error) {
	const defaultDuration = time.Hour
	if ctx != nil && ctx.Message != nil && ctx.Message.ReplyToMessage != nil && ctx.Message.ReplyToMessage.From != nil {
		args := commandArgs(ctx)
		duration := defaultDuration
		if len(args) > 0 {
			if d, err := parseModerationDuration(args[0]); err == nil {
				duration = d
			}
		}
		return []int64{ctx.Message.ReplyToMessage.From.Id}, duration, nil
	}
	args := commandArgs(ctx)
	if len(args) == 0 {
		return nil, 0, fmt.Errorf("missing targets")
	}
	var ids []int64
	i := 0
	for i < len(args) {
		id, err := strconv.ParseInt(args[i], 10, 64)
		if err != nil || id == 0 {
			break
		}
		ids = append(ids, id)
		i++
	}
	if len(ids) == 0 {
		return nil, 0, fmt.Errorf("no valid user IDs")
	}
	duration := defaultDuration
	if i < len(args) {
		if d, err := parseModerationDuration(args[i]); err == nil {
			duration = d
		}
	}
	return ids, duration, nil
}

func batchResultMessage(action string, successes, failures []string) string {
	var sb strings.Builder
	if len(successes) > 0 {
		sb.WriteString(fmt.Sprintf("已%s %d 人：%s。\n", action, len(successes), strings.Join(successes, ", ")))
	}
	if len(failures) > 0 {
		sb.WriteString(fmt.Sprintf("失败 %d 项：%s", len(failures), strings.Join(failures, "、")))
	}
	return strings.TrimSpace(sb.String())
}

func (a *App) withModerationTarget(b *gotgbot.Bot, ctx *ext.Context, action string, fn func(RequestScope, int64, string) error) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, args, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, fmt.Sprintf("用法：回复目标消息 /%s 原因，或 /%s <user_id> 原因。", strings.ToLower(action), strings.ToLower(action)), nil)
	}
	return fn(scope, targetID, strings.TrimSpace(strings.Join(args, " ")))
}

func moderationTarget(ctx *ext.Context) (int64, []string, error) {
	args := commandArgs(ctx)
	if ctx != nil && ctx.Message != nil && ctx.Message.ReplyToMessage != nil && ctx.Message.ReplyToMessage.From != nil {
		return ctx.Message.ReplyToMessage.From.Id, args, nil
	}
	if len(args) == 0 {
		return 0, nil, fmt.Errorf("missing target")
	}
	if strings.HasPrefix(args[0], "@") {
		return 0, nil, fmt.Errorf("username lookup is not available")
	}
	userID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || userID == 0 {
		return 0, nil, fmt.Errorf("invalid target")
	}
	return userID, args[1:], nil
}

func parseModerationDuration(raw string) (time.Duration, error) {
	if strings.HasSuffix(raw, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid days")
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid duration")
	}
	return duration, nil
}

func mutePermissions() gotgbot.ChatPermissions {
	return gotgbot.ChatPermissions{}
}

func fullPermissions() gotgbot.ChatPermissions {
	yes := true
	return gotgbot.ChatPermissions{
		CanSendMessages:       true,
		CanSendAudios:         true,
		CanSendDocuments:      true,
		CanSendPhotos:         true,
		CanSendVideos:         true,
		CanSendVideoNotes:     true,
		CanSendVoiceNotes:     true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
		CanReactToMessages:    &yes,
		CanChangeInfo:         true,
		CanInviteUsers:        true,
		CanPinMessages:        true,
		CanManageTopics:       &yes,
	}
}

func (a *App) handleBanGhosts(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组管理服务尚未接入。", nil)
	}
	userIDs, err := a.services.Admin.ListSeenUsers(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return sendText(b, ctx, "没有记录到任何成员，请等待新成员加入后再试。", nil)
	}

	const checkLimit = 500
	if len(userIDs) > checkLimit {
		userIDs = userIDs[:checkLimit]
	}

	var banned []string
	checked := 0
	for _, uid := range userIDs {
		member, err := b.GetChatMemberWithContext(scope.Context, scope.Chat.ID, uid, nil)
		if err != nil {
			continue
		}
		checked++
		user := member.GetUser()
		if user.FirstName == "" && !user.IsBot {
			if _, banErr := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, uid, nil); banErr == nil {
				banned = append(banned, fmt.Sprintf("%d", uid))
				a.auditModAction(scope.Actor.ID, scope.Chat.ID, "ban_ghost", uid, "deleted account")
			}
		}
	}

	if len(banned) == 0 {
		return sendText(b, ctx, fmt.Sprintf("已检查 %d 人，未发现注销账号。", checked), nil)
	}
	return sendText(b, ctx, fmt.Sprintf("已清理 %d 个注销账号：%s", len(banned), strings.Join(banned, ", ")), nil)
}

func (a *App) handleReport(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if ctx.Message == nil || ctx.Message.ReplyToMessage == nil || ctx.Message.ReplyToMessage.From == nil {
		return sendText(b, ctx, "用法：回复目标消息 /report。", nil)
	}
	target := ctx.Message.ReplyToMessage.From

	// 5-minute cooldown
	if a.services.Redis != nil {
		cooldownKey := fmt.Sprintf("report:cooldown:%d:%d", scope.Chat.ID, scope.Actor.ID)
		if raw, _ := a.services.Redis.Get(scope.Context, cooldownKey).Result(); raw != "" {
			return sendText(b, ctx, "你举报太频繁了，请 5 分钟后再试。", nil)
		}
		_ = a.services.Redis.Set(scope.Context, cooldownKey, "1", 5*time.Minute)
	}

	admins, err := a.getCachedAdmins(scope.Context, b, scope.Chat.ID)
	if err != nil {
		return sendText(b, ctx, "获取管理员列表失败，请稍后重试。", nil)
	}

	if a.isAdminInCache(admins, target.Id) {
		return sendText(b, ctx, "不能举报管理员。", nil)
	}

	var mentions []string
	for _, admin := range admins {
		if admin.IsBot || len(mentions) >= 5 {
			continue
		}
		if admin.Username != "" {
			mentions = append(mentions, "@"+admin.Username)
		} else {
			mentions = append(mentions, fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`,
				admin.UserID, html.EscapeString(admin.FirstName)))
		}
	}

	reporterName := scope.Actor.FirstName
	if scope.Actor.Username != "" {
		reporterName = "@" + scope.Actor.Username
	}
	targetName := target.FirstName
	if target.Username != "" {
		targetName = "@" + target.Username
	}

	text := fmt.Sprintf(
		"🚨 举报通知\n%s 举报了 %s 的消息。\n\n管理员：%s",
		html.EscapeString(reporterName),
		html.EscapeString(targetName),
		strings.Join(mentions, " "),
	)
	if err := sendText(b, ctx, text, &gotgbot.SendMessageOpts{ParseMode: "HTML"}); err != nil {
		return err
	}
	_, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, ctx.Message.MessageId, nil)
	return nil
}

func (a *App) handlePromote(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, _, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标用户 /promote，或 /promote <user_id>。", nil)
	}
	yes := true
	if _, err := b.PromoteChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.PromoteChatMemberOpts{
		CanManageChat:      true,
		CanDeleteMessages:  true,
		CanRestrictMembers: &yes,
		CanInviteUsers:     true,
		CanPinMessages:     true,
	}); err != nil {
		return err
	}
	a.invalidateAdminCache(scope.Chat.ID)
	a.auditModAction(scope.Actor.ID, scope.Chat.ID, "promote", targetID, "")
	return sendText(b, ctx, fmt.Sprintf("已提升用户 %d 为管理员。", targetID), nil)
}

func (a *App) handleDemote(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, _, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标用户 /demote，或 /demote <user_id>。", nil)
	}
	if _, err := b.PromoteChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.PromoteChatMemberOpts{}); err != nil {
		return err
	}
	a.invalidateAdminCache(scope.Chat.ID)
	a.auditModAction(scope.Actor.ID, scope.Chat.ID, "demote", targetID, "")
	return sendText(b, ctx, fmt.Sprintf("已撤销用户 %d 的管理员权限。", targetID), nil)
}

func (a *App) handleSetTitle(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	targetID, args, err := moderationTarget(ctx)
	if err != nil || len(args) == 0 {
		return sendText(b, ctx, "用法：/set_title <user_id> 头衔，或回复目标用户 /set_title 头衔。", nil)
	}
	title := strings.TrimSpace(strings.Join(args, " "))
	if title == "" {
		return sendText(b, ctx, "头衔不能为空。", nil)
	}
	if _, err := b.SetChatAdministratorCustomTitleWithContext(scope.Context, scope.Chat.ID, targetID, title, nil); err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("已设置用户 %d 的管理员头衔：%s", targetID, title), nil)
}

func (a *App) kickUnverifiedMember(b *gotgbot.Bot, chatID int64, userID int64) error {
	ctx := context.Background()
	if _, err := b.BanChatMemberWithContext(ctx, chatID, userID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
		return err
	}
	_, err := b.UnbanChatMemberWithContext(ctx, chatID, userID, &gotgbot.UnbanChatMemberOpts{})
	_ = a.clearUnverifiedKey(chatID, userID)
	return err
}


func (a *App) showBansPanel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return respondText(b, ctx, "封禁记录服务尚未接入。", moderationAssetsMarkup())
	}
	bans, err := a.services.Admin.ListBans(scope.Context, scope.Chat.ID, 10)
	if err != nil {
		return err
	}
	var builder strings.Builder
	builder.WriteString("🚫 封禁记录面板\n\n")
	builder.WriteString("这里可以回看最近的封禁、解封结果，适合处理申诉和复盘误封。\n")
	if len(bans) == 0 {
		builder.WriteString("当前群还没有封禁记录。\n")
	} else {
		builder.WriteString("最近记录：")
		for i, ban := range bans {
			status := "封禁中"
			if ban.UnbannedAt != nil {
				status = "已解封"
			}
			reason := strings.TrimSpace(ban.Reason)
			if reason == "" {
				reason = "无"
			}
			builder.WriteString(fmt.Sprintf("\n%d. [%s] 用户 %d，原因：%s", i+1, status, ban.UserID, truncateRunes(reason, 28)))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n快捷命令\n/bans\n/unban 用户ID")
	return respondText(b, ctx, builder.String(), moderationAssetsMarkup())
}

func (a *App) showViolationsPanel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Violations == nil {
		return respondText(b, ctx, "违规记录服务尚未接入。", moderationAssetsMarkup())
	}
	records, err := a.services.Violations.ListViolations(scope.Context, scope.Chat.ID, 0, 10, 0)
	if err != nil {
		return err
	}
	text := strings.Join([]string{
		"⚠️ 违规记录面板",
		"",
		"这里会汇总关键词拦截、反垃圾命中和人工处理的记录。",
		formatViolationRecords(records),
		"",
		"快捷命令",
		"/violations [用户ID]",
		"/resolve_violation ID 备注",
		"/ignore_violation ID 备注",
	}, "\n")
	return respondText(b, ctx, text, moderationAssetsMarkup())
}
