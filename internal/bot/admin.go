package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

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
	return a.withModerationTarget(b, ctx, "封禁", func(scope RequestScope, targetID int64, reason string) error {
		if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		if a.services.Admin != nil {
			if err := a.services.Admin.RecordBan(scope.Context, scope.Chat.ID, targetID, scope.Actor.ID, reason); err != nil {
				return err
			}
		}
		if a.services.AuditLog != nil {
			a.services.AuditLog.Log(AuditLogEntry{ActorTelegramID: scope.Actor.ID, ChatTelegramID: scope.Chat.ID, Action: "ban", EntityType: "user", TargetTelegramID: targetID, Detail: reason})
		}
		return sendText(b, ctx, fmt.Sprintf("已封禁用户 %d。%s", targetID, reasonSuffix(reason)), nil)
	})
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
	targetID, args, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：回复目标消息 /mute 30m，或 /mute <user_id> 30m。", nil)
	}
	duration := time.Hour
	if len(args) > 0 {
		duration, err = parseModerationDuration(args[0])
		if err != nil {
			return sendText(b, ctx, "时长格式示例：30m / 2h / 1d。", nil)
		}
	}
	until := time.Now().Add(duration).Unix()
	if _, err := b.RestrictChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, mutePermissions(), &gotgbot.RestrictChatMemberOpts{UntilDate: until, UseIndependentChatPermissions: true}); err != nil {
		return err
	}
	a.auditModAction(scope.Actor.ID, scope.Chat.ID, "mute", targetID, fmt.Sprintf("duration: %s", duration.Round(time.Second)))
	return sendText(b, ctx, fmt.Sprintf("已禁言用户 %d，时长 %s。", targetID, duration.Round(time.Second)), nil)
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
	return a.withModerationTarget(b, ctx, "踢出", func(scope RequestScope, targetID int64, reason string) error {
		if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		if _, err := b.UnbanChatMemberWithContext(scope.Context, scope.Chat.ID, targetID, &gotgbot.UnbanChatMemberOpts{}); err != nil {
			return err
		}
		return sendText(b, ctx, fmt.Sprintf("已踢出用户 %d。%s", targetID, reasonSuffix(reason)), nil)
	})
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

func (a *App) handleVerifyToggle(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	cfg, err := a.services.Admin.ToggleVerify(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	return sendText(b, ctx, "入群验证已切换。\n"+formatAdminConfig(cfg), nil)
}

func (a *App) handleSetVerify(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	args := commandArgs(ctx)
	if len(args) < 1 {
		return sendText(b, ctx, "用法：/set_verify <type|question|options|answer|difficulty> [value]\n\n"+
			"type - 设置验证类型：button, captcha, multi_choice, poll, math\n"+
			"question - 设置验证问题\n"+
			"options - 设置选项（JSON数组，如 [\"A\",\"B\",\"C\"]）\n"+
			"answer - 设置正确答案索引（0开始）\n"+
			"difficulty - 设置验证难度：easy, medium, hard\n\n"+
			"示例：\n"+
			"/set_verify type multi_choice\n"+
			"/set_verify question 群规第一条是什么？\n"+
			"/set_verify options [\"尊重他人\",\"禁止广告\",\"禁止NSFW\",\"以上都是\"]\n"+
			"/set_verify answer 3\n"+
			"/set_verify difficulty hard", nil)
	}
	sub := args[0]
	val := ""
	if len(args) > 1 {
		val = strings.Join(args[1:], " ")
	}
	var patch ChatAdminConfigPatch
	switch sub {
	case "type":
		val = strings.TrimSpace(val)
		if val != "button" && val != "captcha" && val != "multi_choice" && val != "poll" && val != "math" {
			return sendText(b, ctx, "验证类型只能是 button、captcha、multi_choice、poll 或 math。", nil)
		}
		patch.VerifyType = &val
	case "question":
		val = strings.TrimSpace(val)
		if val == "" {
			return sendText(b, ctx, "验证问题不能为空。", nil)
		}
		patch.VerifyQuestion = &val
	case "options":
		val = strings.TrimSpace(val)
		var opts []string
		if err := json.Unmarshal([]byte(val), &opts); err != nil || len(opts) == 0 {
			return sendText(b, ctx, "选项格式无效，请使用 JSON 数组，如 [\"A\",\"B\",\"C\"]。", nil)
		}
		patch.VerifyOptions = &val
	case "answer":
		idx, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil || idx < 0 {
			return sendText(b, ctx, "正确答案索引必须是 >=0 的整数。", nil)
		}
		patch.VerifyCorrectIndex = &idx
	case "difficulty":
		val = strings.TrimSpace(val)
		if val != "easy" && val != "medium" && val != "hard" {
			return sendText(b, ctx, "难度只能是 easy、medium 或 hard。", nil)
		}
		patch.VerifyDifficulty = &val
	default:
		return sendText(b, ctx, "未知子命令："+sub+"。可用：type, question, options, answer, difficulty。", nil)
	}
	cfg, err := a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID, patch)
	if err != nil {
		return err
	}
	return sendText(b, ctx, "验证配置已更新。\n"+formatAdminConfig(cfg), nil)
}

func (a *App) handleVerifyStats(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	stats, err := a.services.Admin.GetVerifyStats(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	text := fmt.Sprintf(
		"📊 验证统计\n━━━━━━━━━━\n通过：%d\n拒绝：%d\n超时：%d\n待验证：%d",
		stats.TotalPassed,
		stats.TotalFailed,
		stats.TotalTimeout,
		stats.PendingCount,
	)
	return sendText(b, ctx, text, nil)
}

func (a *App) handleNewChatMembers(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.Type != "group" && scope.Chat.Type != "supergroup" {
		return nil
	}
	if ctx == nil || ctx.Message == nil || a.services.Admin == nil {
		return nil
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	for _, member := range ctx.Message.NewChatMembers {
		if member.IsBot {
			continue
		}
		// Whitelist check: skip verification for whitelisted users
		if isInWhitelist(cfg.VerifyWhitelist, member.Id) {
			if err := a.sendWelcomeMessage(b, ctx, cfg, member); err != nil {
				return err
			}
			continue
		}
		if !cfg.VerifyEnabled {
			continue
		}
		_ = a.restrictForVerification(b, scope, member.Id)
		// Mark user as unverified in Redis for link/media blocking.
		if a.services.Redis != nil {
			ttl := time.Duration(cfg.VerifyTimeout) * time.Second
			if ttl <= 0 {
				ttl = time.Minute
			}
			_ = a.services.Redis.Set(scope.Context, fmt.Sprintf("unverified:%d:%d", scope.Chat.ID, member.Id), "1", ttl).Err()
		}
		switch cfg.VerifyType {
		case "multi_choice":
			if err := a.sendMultiChoiceChallenge(b, ctx, cfg, member); err != nil {
				return err
			}
		case "poll":
			if err := a.sendPollChallenge(b, ctx, cfg, member); err != nil {
				return err
			}
		case "math":
			if err := a.sendMathChallenge(b, ctx, cfg, member); err != nil {
				return err
			}
		default:
			if err := a.sendVerifyChallenge(b, ctx, cfg, member); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) routeAdminCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "check_admin":
		return a.checkBotAdmin(b, ctx)
	case "config":
		return a.showAdminConfigFromCallback(b, ctx)
	case "verify":
		return a.handleVerifyCallback(b, ctx, payload)
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

func (a *App) showVerifyMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Admin == nil {
		return respondText(b, ctx, "群组配置服务尚未接入。", groupMarkup())
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	text := fmt.Sprintf("✅ 入群验证\n━━━━━━━━━━\n当前状态：%s\n超时时间：%d 秒\n\n开启后，新成员会先被限制发言，需要完成按钮验证才会恢复权限。", boolLabel(cfg.VerifyEnabled, "开启", "关闭"), cfg.VerifyTimeout)
	return respondText(b, ctx, text, verifyMenuMarkup())
}

func (a *App) toggleVerifyFromCallback(b *gotgbot.Bot, ctx *ext.Context) error {
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	scope := requestScope(ctx)
	cfg, err := a.services.Admin.ToggleVerify(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	text := fmt.Sprintf("入群验证已切换。\n状态：%s\n超时：%d 秒", boolLabel(cfg.VerifyEnabled, "开启", "关闭"), cfg.VerifyTimeout)
	return respondText(b, ctx, text, verifyMenuMarkup())
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

func (a *App) handleVerifyCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if ctx == nil || ctx.CallbackQuery == nil || len(payload.Arguments) < 2 || a.services.Admin == nil {
		return answerCallback(b, ctx, "验证已失效")
	}
	userID, err := strconv.ParseInt(payload.Resource, 10, 64)
	if err != nil {
		return answerCallback(b, ctx, "验证数据无效")
	}
	if ctx.CallbackQuery.From.Id != userID {
		return answerCallback(b, ctx, "这不是你的验证")
	}
	chatID, err := strconv.ParseInt(payload.Arguments[0], 10, 64)
	if err != nil {
		return answerCallback(b, ctx, "验证数据无效")
	}
	answer := payload.Arguments[1]
	result, err := a.services.Admin.CheckVerifyChallenge(requestScope(ctx).Context, chatID, userID, answer)
	if err != nil {
		return err
	}
	if !result.OK {
		if result.ShouldKick {
			_ = a.kickUnverifiedMember(b, chatID, userID)
			_ = a.clearUnverifiedKey(chatID, userID)
			if result.Challenge.MessageID != 0 {
				_, _ = b.DeleteMessageWithContext(requestScope(ctx).Context, chatID, result.Challenge.MessageID, nil)
			}
			return answerCallback(b, ctx, "验证失败，已移出群组")
		}
		if result.Expired {
			return answerCallback(b, ctx, "验证已超时")
		}
		return answerCallback(b, ctx, fmt.Sprintf("答案不对，还剩 %d 次", result.RemainingAttempts))
	}
	if _, err := b.RestrictChatMemberWithContext(requestScope(ctx).Context, chatID, userID, fullPermissions(), &gotgbot.RestrictChatMemberOpts{UseIndependentChatPermissions: true}); err != nil {
		return err
	}
	if ctx.CallbackQuery.Message != nil {
		_, _ = ctx.CallbackQuery.Message.Delete(b, nil)
	}
	_ = answerCallback(b, ctx, "验证通过")
	_ = a.clearUnverifiedKey(chatID, userID)
	cfg, _ := a.services.Admin.GetConfig(requestScope(ctx).Context, chatID)
	name := strings.TrimSpace(ctx.CallbackQuery.From.FirstName + " " + ctx.CallbackQuery.From.LastName)
	if name == "" {
		name = fmt.Sprintf("%d", userID)
	}
	text := strings.ReplaceAll(cfg.WelcomeText, "{name}", name)
	welcome, err := b.SendMessageWithContext(requestScope(ctx).Context, chatID, text, nil)
	if err == nil && welcome != nil {
		go deleteMessageLater(b, chatID, welcome.MessageId, 30*time.Second)
	}
	return err
}

func (a *App) handleVerifyAnswer(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if ctx == nil || ctx.CallbackQuery == nil || len(payload.Arguments) < 2 || a.services.Admin == nil {
		return answerCallback(b, ctx, "验证已失效")
	}
	// payload format: domain="verify", action="answer", resource=chatID, arguments=[userID, optionIndex]
	chatID, err := strconv.ParseInt(payload.Resource, 10, 64)
	if err != nil {
		return answerCallback(b, ctx, "验证数据无效")
	}
	userID, err := strconv.ParseInt(payload.Arguments[0], 10, 64)
	if err != nil {
		return answerCallback(b, ctx, "验证数据无效")
	}
	if ctx.CallbackQuery.From.Id != userID {
		return answerCallback(b, ctx, "这不是你的验证")
	}
	optionIndex, err := strconv.Atoi(payload.Arguments[1])
	if err != nil {
		return answerCallback(b, ctx, "验证数据无效")
	}

	// Get the stored challenge to check the correct answer index
	answer := strconv.Itoa(optionIndex)
	result, err := a.services.Admin.CheckVerifyChallenge(requestScope(ctx).Context, chatID, userID, answer)
	if err != nil {
		return err
	}
	if !result.OK {
		if result.ShouldKick {
			_ = a.kickUnverifiedMember(b, chatID, userID)
			_ = a.clearUnverifiedKey(chatID, userID)
			if result.Challenge.MessageID != 0 {
				_, _ = b.DeleteMessageWithContext(requestScope(ctx).Context, chatID, result.Challenge.MessageID, nil)
			}
			_ = a.services.Admin.RecordVerifyEvent(requestScope(ctx).Context, chatID, userID, "verify_fail", "答案错误超过次数被踢出")
			return answerCallback(b, ctx, "验证失败，已移出群组")
		}
		if result.Expired {
			return answerCallback(b, ctx, "验证已超时")
		}
		return answerCallback(b, ctx, fmt.Sprintf("答案不对，还剩 %d 次", result.RemainingAttempts))
	}
	// Correct answer
	if _, err := b.RestrictChatMemberWithContext(requestScope(ctx).Context, chatID, userID, fullPermissions(), &gotgbot.RestrictChatMemberOpts{UseIndependentChatPermissions: true}); err != nil {
		return err
	}
	if ctx.CallbackQuery.Message != nil {
		_, _ = ctx.CallbackQuery.Message.Delete(b, nil)
	}
	_ = answerCallback(b, ctx, "验证通过")
	_ = a.clearUnverifiedKey(chatID, userID)
	_ = a.services.Admin.RecordVerifyEvent(requestScope(ctx).Context, chatID, userID, "verify_pass", "选择题验证通过")
	cfg, _ := a.services.Admin.GetConfig(requestScope(ctx).Context, chatID)
	name := strings.TrimSpace(ctx.CallbackQuery.From.FirstName + " " + ctx.CallbackQuery.From.LastName)
	if name == "" {
		name = fmt.Sprintf("%d", userID)
	}
	text := strings.ReplaceAll(cfg.WelcomeText, "{name}", name)
	welcome, err := b.SendMessageWithContext(requestScope(ctx).Context, chatID, text, nil)
	if err == nil && welcome != nil {
		go deleteMessageLater(b, chatID, welcome.MessageId, 30*time.Second)
	}
	return err
}

func (a *App) routeVerifyCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "answer":
		return a.handleVerifyAnswer(b, ctx, payload)
	default:
		return answerCallback(b, ctx, "未知验证操作")
	}
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

func (a *App) restrictForVerification(b *gotgbot.Bot, scope RequestScope, userID int64) error {
	_, err := b.RestrictChatMemberWithContext(scope.Context, scope.Chat.ID, userID, mutePermissions(), &gotgbot.RestrictChatMemberOpts{UseIndependentChatPermissions: true})
	return err
}

func (a *App) sendVerifyChallenge(b *gotgbot.Bot, ctx *ext.Context, cfg ChatAdminConfig, user gotgbot.User) error {
	left := rand.Intn(8) + 2
	right := rand.Intn(8) + 1
	answer := left + right
	timeout := time.Duration(cfg.VerifyTimeout) * time.Second
	if timeout <= 0 {
		timeout = time.Minute
	}
	maxAttempts := 3
	numOptions := defaultNumOptions()

	// Apply difficulty settings
	switch cfg.VerifyDifficulty {
	case "easy":
		timeout = 120 * time.Second
		maxAttempts = 5
		numOptions = 2
	case "hard":
		timeout = 30 * time.Second
		maxAttempts = 2
		numOptions = 5
	}

	options := buildMathOptions(answer, numOptions)
	rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
	buttons := make([]gotgbot.InlineKeyboardButton, 0, len(options))
	for _, option := range options {
		buttons = append(buttons, gotgbot.InlineKeyboardButton{
			Text:         strconv.Itoa(option),
			CallbackData: CallbackData("admin", "verify", strconv.FormatInt(user.Id, 10), strconv.FormatInt(cfg.ChatID, 10), strconv.Itoa(option)),
		})
	}

	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = strconv.FormatInt(user.Id, 10)
	}
	question := fmt.Sprintf("%d + %d = ?", left, right)
	sent, err := b.SendMessageWithContext(requestScope(ctx).Context, cfg.ChatID, fmt.Sprintf("%s，请完成验证：%s", name, question), &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{buttons}},
	})
	if err != nil {
		return err
	}
	messageID := int64(0)
	if sent != nil {
		messageID = sent.MessageId
	}
	if err := a.services.Admin.SetVerifyChallenge(requestScope(ctx).Context, cfg.ChatID, user.Id, VerifyChallenge{
		Answer:     strconv.Itoa(answer),
		MessageID:  messageID,
		Attempts:   maxAttempts,
		ExpireAt:   time.Now().Add(timeout),
		Question:   question,
		MemberName: name,
	}, timeout); err != nil {
		return err
	}
	return nil
}

func (a *App) sendMultiChoiceChallenge(b *gotgbot.Bot, ctx *ext.Context, cfg ChatAdminConfig, user gotgbot.User) error {
	timeout := time.Duration(cfg.VerifyTimeout) * time.Second
	if timeout <= 0 {
		timeout = time.Minute
	}
	maxAttempts := 3

	// Apply difficulty settings
	switch cfg.VerifyDifficulty {
	case "easy":
		timeout = 120 * time.Second
		maxAttempts = 5
	case "hard":
		timeout = 30 * time.Second
		maxAttempts = 2
	}

	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = strconv.FormatInt(user.Id, 10)
	}

	question := cfg.VerifyQuestion
	if question == "" {
		question = "请回答以下验证问题："
	}

	var options []string
	if err := json.Unmarshal([]byte(cfg.VerifyOptions), &options); err != nil || len(options) == 0 {
		return fmt.Errorf("验证选项未配置或格式无效")
	}

	// For easy difficulty, only use first 2 options
	if cfg.VerifyDifficulty == "easy" && len(options) > 2 {
		// Ensure correct answer is within the first 2 options and track its new index
		correctIdx := cfg.VerifyCorrectIndex
		if correctIdx < 0 || correctIdx >= len(options) {
			correctIdx = 0
		}
		if correctIdx >= 2 {
			// Swap correct answer into position 0
			options[0], options[correctIdx] = options[correctIdx], options[0]
			correctIdx = 0
		}
		options = options[:2]
		cfg.VerifyCorrectIndex = correctIdx
	}

	buttons := make([]gotgbot.InlineKeyboardButton, 0, len(options))
	for i, option := range options {
		buttons = append(buttons, gotgbot.InlineKeyboardButton{
			Text:         option,
			CallbackData: CallbackData("verify", "answer", strconv.FormatInt(cfg.ChatID, 10), strconv.FormatInt(user.Id, 10), strconv.Itoa(i)),
		})
	}

	sent, err := b.SendMessageWithContext(requestScope(ctx).Context, cfg.ChatID, fmt.Sprintf("%s，请完成验证：\n\n%s", name, question), &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{buttons}},
	})
	if err != nil {
		return err
	}
	messageID := int64(0)
	if sent != nil {
		messageID = sent.MessageId
	}
	// Determine correct answer index after possible trimming
	correctAnswer := "-1"
	if cfg.VerifyCorrectIndex >= 0 && cfg.VerifyCorrectIndex < len(options) {
		correctAnswer = strconv.Itoa(cfg.VerifyCorrectIndex)
	}
	if err := a.services.Admin.SetVerifyChallenge(requestScope(ctx).Context, cfg.ChatID, user.Id, VerifyChallenge{
		Answer:     correctAnswer,
		MessageID:  messageID,
		Attempts:   maxAttempts,
		ExpireAt:   time.Now().Add(timeout),
		Question:   question,
		MemberName: name,
	}, timeout); err != nil {
		return err
	}
	return nil
}

func (a *App) sendPollChallenge(b *gotgbot.Bot, ctx *ext.Context, cfg ChatAdminConfig, user gotgbot.User) error {
	timeout := time.Duration(cfg.VerifyTimeout) * time.Second
	if timeout <= 0 {
		timeout = time.Minute
	}

	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = strconv.FormatInt(user.Id, 10)
	}

	question := cfg.VerifyQuestion
	if question == "" {
		question = "请回答验证问题"
	}

	var options []string
	if err := json.Unmarshal([]byte(cfg.VerifyOptions), &options); err != nil || len(options) < 2 {
		return fmt.Errorf("投票验证至少需要2个选项")
	}

	// Determine correct answer index
	correctAnswerIdx := cfg.VerifyCorrectIndex
	if correctAnswerIdx < 0 || correctAnswerIdx >= len(options) {
		correctAnswerIdx = 0
	}

	// Telegram quiz polls require at least 2 options, max 10
	if len(options) > 10 {
		options = options[:10]
	}

	pollQuestion := fmt.Sprintf("%s，%s", name, question)
	sent, err := pollOpts := make([]gotgbot.InputPollOption, len(options))
	for i, opt := range options {
		pollOpts[i] = gotgbot.InputPollOption{Text: opt}
	}
	anonymous := false
	sent, err := b.SendPollWithContext(requestScope(ctx).Context, cfg.ChatID, pollQuestion, pollOpts, &gotgbot.SendPollOpts{
		Type:            "quiz",
		CorrectOptionId: int64(correctAnswerIdx),
		IsAnonymous:     &anonymous,
		Explanation:     "选择正确答案即可入群",
	})
	if err != nil {
		return err
	}
	pollID := ""
	messageID := int64(0)
	if sent != nil {
		pollID = sent.Poll.Id
		messageID = sent.MessageId
	}
	if err := a.services.Admin.SetVerifyChallenge(requestScope(ctx).Context, cfg.ChatID, user.Id, VerifyChallenge{
		Answer:     strconv.Itoa(correctAnswerIdx),
		MessageID:  messageID,
		Attempts:   3,
		ExpireAt:   time.Now().Add(timeout),
		Question:   question,
		MemberName: name,
		PollID:     pollID,
	}, timeout); err != nil {
		return err
	}
	return nil
}

func (a *App) sendMathChallenge(b *gotgbot.Bot, ctx *ext.Context, cfg ChatAdminConfig, user gotgbot.User) error {
	timeout := time.Duration(cfg.VerifyTimeout) * time.Second
	if timeout <= 0 {
		timeout = time.Minute
	}
	maxAttempts := 3
	numOptions := 4

	// Apply difficulty settings
	switch cfg.VerifyDifficulty {
	case "easy":
		timeout = 120 * time.Second
		maxAttempts = 5
		numOptions = 2
	case "hard":
		timeout = 30 * time.Second
		maxAttempts = 2
		numOptions = 5
	}

	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = strconv.FormatInt(user.Id, 10)
	}

	question, answer := generateMathQuestion()

	// Generate options: correct answer + distractors
	options := buildMathOptions(answer, numOptions)
	rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })

	buttons := make([]gotgbot.InlineKeyboardButton, 0, len(options))
	for _, option := range options {
		buttons = append(buttons, gotgbot.InlineKeyboardButton{
			Text:         strconv.Itoa(option),
			CallbackData: CallbackData("verify", "answer", strconv.FormatInt(cfg.ChatID, 10), strconv.FormatInt(user.Id, 10), strconv.Itoa(option)),
		})
	}

	sent, err := b.SendMessageWithContext(requestScope(ctx).Context, cfg.ChatID, fmt.Sprintf("%s，请完成验证：\n\n%s", name, question), &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{buttons}},
	})
	if err != nil {
		return err
	}
	messageID := int64(0)
	if sent != nil {
		messageID = sent.MessageId
	}
	if err := a.services.Admin.SetVerifyChallenge(requestScope(ctx).Context, cfg.ChatID, user.Id, VerifyChallenge{
		Answer:     strconv.Itoa(answer),
		MessageID:  messageID,
		Attempts:   maxAttempts,
		ExpireAt:   time.Now().Add(timeout),
		Question:   question,
		MemberName: name,
	}, timeout); err != nil {
		return err
	}
	return nil
}

func generateMathQuestion() (question string, correctAnswer int) {
	a := rand.Intn(10) + 1
	b := rand.Intn(10) + 1
	op := []string{"+", "-", "×"}[rand.Intn(3)]
	switch op {
	case "+":
		correctAnswer = a + b
	case "-":
		correctAnswer = a - b
	case "×":
		correctAnswer = a * b
	}
	return fmt.Sprintf("%d %s %d = ?", a, op, b), correctAnswer
}

func (a *App) handlePollAnswer(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx == nil || ctx.PollAnswer == nil || a.services.Admin == nil {
		return nil
	}
	if ctx.EffectiveChat == nil || ctx.EffectiveUser == nil {
		return nil
	}
	chatID := ctx.EffectiveChat.Id
	userID := ctx.EffectiveUser.Id
	if chatID == 0 || userID == 0 {
		return nil
	}

	// Only handle group/supergroup polls
	if ctx.EffectiveChat.Type != "group" && ctx.EffectiveChat.Type != "supergroup" {
		return nil
	}

	// Get the challenge for this user
	challenge, ok, err := a.services.Admin.GetVerifyChallenge(requestScope(ctx).Context, chatID, userID)
	if err != nil || !ok {
		return nil
	}

	// Only process poll answers that match a stored poll challenge
	if challenge.PollID == "" || ctx.PollAnswer.PollId != challenge.PollID {
		return nil
	}

	// Get the selected option index
	if len(ctx.PollAnswer.OptionIds) == 0 {
		return nil
	}
	selectedOption := int(ctx.PollAnswer.OptionIds[0])

	answer := strconv.Itoa(selectedOption)
	result, err := a.services.Admin.CheckVerifyChallenge(requestScope(ctx).Context, chatID, userID, answer)
	if err != nil {
		return err
	}

	if !result.OK {
		if result.ShouldKick {
			_ = a.kickUnverifiedMember(b, chatID, userID)
			_ = a.clearUnverifiedKey(chatID, userID)
			if result.Challenge.MessageID != 0 {
				_, _ = b.DeleteMessageWithContext(requestScope(ctx).Context, chatID, result.Challenge.MessageID, nil)
			}
			_ = a.services.Admin.RecordVerifyEvent(requestScope(ctx).Context, chatID, userID, "verify_fail", "投票验证错误超过次数被踢出")
			return nil
		}
		// Wrong answer but still has attempts
		return nil
	}

	// Correct answer - restore permissions
	if _, err := b.RestrictChatMemberWithContext(requestScope(ctx).Context, chatID, userID, fullPermissions(), &gotgbot.RestrictChatMemberOpts{UseIndependentChatPermissions: true}); err != nil {
		return err
	}
	_ = a.clearUnverifiedKey(chatID, userID)
	_ = a.services.Admin.RecordVerifyEvent(requestScope(ctx).Context, chatID, userID, "verify_pass", "投票验证通过")
	cfg, _ := a.services.Admin.GetConfig(requestScope(ctx).Context, chatID)
	name := strings.TrimSpace(ctx.EffectiveUser.FirstName + " " + ctx.EffectiveUser.LastName)
	if name == "" {
		name = fmt.Sprintf("%d", userID)
	}
	text := strings.ReplaceAll(cfg.WelcomeText, "{name}", name)
	welcome, err := b.SendMessageWithContext(requestScope(ctx).Context, chatID, text, nil)
	if err == nil && welcome != nil {
		go deleteMessageLater(b, chatID, welcome.MessageId, 30*time.Second)
	}
	return nil
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

// clearUnverifiedKey removes the unverified marker from Redis.
func (a *App) clearUnverifiedKey(chatID int64, userID int64) error {
	if a.services.Redis == nil {
		return nil
	}
	key := fmt.Sprintf("unverified:%d:%d", chatID, userID)
	return a.services.Redis.Del(context.Background(), key).Err()
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

func isInWhitelist(whitelist string, userID int64) bool {
	if whitelist == "" {
		return false
	}
	uid := strconv.FormatInt(userID, 10)
	for _, entry := range strings.Split(whitelist, ",") {
		if strings.TrimSpace(entry) == uid {
			return true
		}
	}
	return false
}

func addToWhitelist(whitelist string, userID int64) string {
	uid := strconv.FormatInt(userID, 10)
	if whitelist == "" {
		return uid
	}
	entries := strings.Split(whitelist, ",")
	for _, entry := range entries {
		if strings.TrimSpace(entry) == uid {
			return whitelist
		}
	}
	return whitelist + "," + uid
}

func removeFromWhitelist(whitelist string, userID int64) string {
	uid := strconv.FormatInt(userID, 10)
	if whitelist == "" {
		return ""
	}
	var kept []string
	for _, entry := range strings.Split(whitelist, ",") {
		e := strings.TrimSpace(entry)
		if e != "" && e != uid {
			kept = append(kept, e)
		}
	}
	return strings.Join(kept, ",")
}

// sendWelcomeMessage sends a welcome message to the new member (without verification)
func (a *App) sendWelcomeMessage(b *gotgbot.Bot, ctx *ext.Context, cfg ChatAdminConfig, user gotgbot.User) error {
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = strconv.FormatInt(user.Id, 10)
	}
	text := strings.ReplaceAll(cfg.WelcomeText, "{name}", name)
	welcome, err := b.SendMessageWithContext(requestScope(ctx).Context, cfg.ChatID, text, nil)
	if err == nil && welcome != nil {
		go deleteMessageLater(b, cfg.ChatID, welcome.MessageId, 30*time.Second)
	}
	return err
}

func (a *App) handleAllowUser(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	targetID, _, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：/allowuser @username 或 /allowuser 123456", nil)
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	if isInWhitelist(cfg.VerifyWhitelist, targetID) {
		return sendText(b, ctx, fmt.Sprintf("用户 %d 已在白名单中。", targetID), nil)
	}
	newWhitelist := addToWhitelist(cfg.VerifyWhitelist, targetID)
	_, err = a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID, ChatAdminConfigPatch{VerifyWhitelist: &newWhitelist})
	if err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("已将用户 %d 加入免验证白名单。", targetID), nil)
}

func (a *App) handleDelAllowUser(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	targetID, _, err := moderationTarget(ctx)
	if err != nil {
		return sendText(b, ctx, "用法：/delallowuser 123456", nil)
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	if !isInWhitelist(cfg.VerifyWhitelist, targetID) {
		return sendText(b, ctx, fmt.Sprintf("用户 %d 不在白名单中。", targetID), nil)
	}
	newWhitelist := removeFromWhitelist(cfg.VerifyWhitelist, targetID)
	_, err = a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID, ChatAdminConfigPatch{VerifyWhitelist: &newWhitelist})
	if err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("已将用户 %d 从免验证白名单移除。", targetID), nil)
}

// defaultNumOptions returns the default number of math challenge options
func defaultNumOptions() int {
	return 4
}

// buildMathOptions builds a slice of unique integer options with the correct answer included
func buildMathOptions(answer int, count int) []int {
	if count < 2 {
		count = 2
	}
	options := []int{answer}
	for len(options) < count {
		offset := rand.Intn(5) + 1
		distractor := answer + offset
		if rand.Intn(2) == 0 {
			distractor = answer - offset
		}
		dupe := false
		for _, o := range options {
			if o == distractor {
				dupe = true
				break
			}
		}
		if !dupe {
			options = append(options, distractor)
		}
	}
	return options
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