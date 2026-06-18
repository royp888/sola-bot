package bot

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/pollanswer"
)

func (a *App) registerVerifyHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("verify_toggle", a.wrap(a.handleVerifyToggle, a.RequirePermission(PermissionVerify), a.RateLimit("cmd:verify_toggle", 1))))
	d.AddHandler(handlers.NewCommand("set_verify", a.wrap(a.handleSetVerify, a.RequirePermission(PermissionVerify), a.RateLimit("cmd:set_verify", 1))))
	d.AddHandler(handlers.NewCommand("verify_stats", a.wrap(a.handleVerifyStats, a.RequirePermission(PermissionVerify), a.RateLimit("cmd:verify_stats", 1))))
	d.AddHandler(handlers.NewCommand("allowuser", a.wrap(a.handleAllowUser, a.RequirePermission(PermissionVerify), a.RateLimit("cmd:allowuser", 1))))
	d.AddHandler(handlers.NewCommand("delallowuser", a.wrap(a.handleDelAllowUser, a.RequirePermission(PermissionVerify), a.RateLimit("cmd:delallowuser", 1))))
	d.AddHandler(handlers.NewMessage(message.NewChatMembers, a.handleNewChatMembers))
	d.AddHandler(handlers.NewChatJoinRequest(filters.ChatJoinRequest(func(_ *gotgbot.ChatJoinRequest) bool { return true }), a.handleChatJoinRequest))
	d.AddHandler(handlers.NewPollAnswer(pollanswer.All, a.handlePollAnswer))
}

func (a *App) handleVerifyToggle(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
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
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	args := commandArgs(ctx)
	if len(args) < 1 {
		return sendText(b, ctx, "用法：/set_verify <type|question|options|answer|difficulty> [value]\n\n"+
			"type - 设置验证类型：button, captcha, multi_choice, poll, math, turnstile\n"+
			"question - 设置验证问题\n"+
			"options - 设置选项（JSON数组，如 [\"A\",\"B\",\"C\"]）\n"+
			"answer - 设置正确答案索引（0开始）\n"+
			"difficulty - 设置验证难度：easy, medium, hard\n\n"+
			"示例：\n"+
			"/set_verify type multi_choice\n"+
			"/set_verify type turnstile  （需配置 SOLA_BOT_MINI_APP_URL 和 SOLA_TURNSTILE_VERIFY_SECRET）\n"+
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
		if val != "button" && val != "captcha" && val != "multi_choice" && val != "poll" && val != "math" && val != "turnstile" {
			return sendText(b, ctx, "验证类型只能是 button、captcha、multi_choice、poll、math 或 turnstile。", nil)
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
		if a.services.Admin != nil {
			_ = a.services.Admin.RecordSeenUser(scope.Context, scope.Chat.ID, member.Id)
		}
		// Skip verification for group owner and administrators.
		// Owners/admins joining through links still trigger new_chat_members,
		// but asking them to verify would loop forever.
		if chatMember, err := b.GetChatMemberWithContext(scope.Context, scope.Chat.ID, member.Id, nil); err == nil {
			switch chatMember.(type) {
			case gotgbot.ChatMemberOwner, gotgbot.ChatMemberAdministrator:
				_ = a.sendWelcomeMessage(b, ctx, cfg, member)
				continue
			}
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
		case "button":
			if err := a.sendButtonChallenge(b, ctx, cfg, member); err != nil {
				return err
			}
		default:
			if err := a.sendButtonChallenge(b, ctx, cfg, member); err != nil {
				return err
			}
		}
	}
	return nil
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

func (a *App) handleVerifyCheck(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if ctx == nil || ctx.CallbackQuery == nil || len(payload.Arguments) < 2 || a.services.Admin == nil {
		return answerCallback(b, ctx, "验证已失效")
	}
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
		// Regular groups don't support restrictChatMember — log and continue so
		// the user still gets the "验证通过" popup and the challenge message is cleaned up.
		log.Printf("restrictChatMember after verify (chat=%d user=%d): %v", chatID, userID, err)
	}
	if ctx.CallbackQuery.Message != nil {
		_, _ = ctx.CallbackQuery.Message.Delete(b, nil)
	}
	_ = answerCallback(b, ctx, "验证通过")
	_ = a.clearUnverifiedKey(chatID, userID)
	cfg, _ := a.services.Admin.GetConfig(requestScope(ctx).Context, chatID)
	return a.postWelcomeMessage(b, ctx, chatID, cfg, ctx.CallbackQuery.From)
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
		log.Printf("restrictChatMember after verify answer (chat=%d user=%d): %v", chatID, userID, err)
	}
	if ctx.CallbackQuery.Message != nil {
		_, _ = ctx.CallbackQuery.Message.Delete(b, nil)
	}
	_ = answerCallback(b, ctx, "验证通过")
	_ = a.clearUnverifiedKey(chatID, userID)
	_ = a.services.Admin.RecordVerifyEvent(requestScope(ctx).Context, chatID, userID, "verify_pass", "选择题验证通过")
	cfg, _ := a.services.Admin.GetConfig(requestScope(ctx).Context, chatID)
	return a.postWelcomeMessage(b, ctx, chatID, cfg, ctx.CallbackQuery.From)
}

func (a *App) routeVerifyCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "check":
		return a.handleVerifyCheck(b, ctx, payload)
	case "answer":
		return a.handleVerifyAnswer(b, ctx, payload)
	default:
		return answerCallback(b, ctx, "未知验证操作")
	}
}

func (a *App) restrictForVerification(b *gotgbot.Bot, scope RequestScope, userID int64) error {
	_, err := b.RestrictChatMemberWithContext(scope.Context, scope.Chat.ID, userID, mutePermissions(), &gotgbot.RestrictChatMemberOpts{UseIndependentChatPermissions: true})
	return err
}

func (a *App) sendButtonChallenge(b *gotgbot.Bot, ctx *ext.Context, cfg ChatAdminConfig, user gotgbot.User) error {
	timeout := time.Duration(cfg.VerifyTimeout) * time.Second
	if timeout <= 0 {
		timeout = time.Minute
	}
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = strconv.FormatInt(user.Id, 10)
	}
	userIDStr := strconv.FormatInt(user.Id, 10)
	chatIDStr := strconv.FormatInt(cfg.ChatID, 10)
	text := fmt.Sprintf("👋 欢迎 %s！\n请在 %d 秒内点击下方按钮完成入群确认。", name, int(timeout.Seconds()))
	sent, err := b.SendMessageWithContext(requestScope(ctx).Context, cfg.ChatID, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "✅ 我已阅读群规，点击入群", CallbackData: CallbackData("verify", "check", chatIDStr, userIDStr, "ok")}},
		}},
	})
	if err != nil {
		return err
	}
	messageID := int64(0)
	if sent != nil {
		messageID = sent.MessageId
	}
	if err := a.services.Admin.SetVerifyChallenge(requestScope(ctx).Context, cfg.ChatID, user.Id, VerifyChallenge{
		Answer:     "ok",
		MessageID:  messageID,
		Attempts:   1,
		ExpireAt:   time.Now().Add(timeout),
		MemberName: name,
	}, timeout); err != nil {
		return err
	}
	a.scheduleVerifyTimeoutKick(b, cfg.ChatID, user.Id, timeout)
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
	a.scheduleVerifyTimeoutKick(b, cfg.ChatID, user.Id, timeout)
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
	pollOpts := make([]gotgbot.InputPollOption, len(options))
	for i, opt := range options {
		pollOpts[i] = gotgbot.InputPollOption{Text: opt}
	}
	anonymous := false
	sent, err := b.SendPollWithContext(requestScope(ctx).Context, cfg.ChatID, pollQuestion, pollOpts, &gotgbot.SendPollOpts{
		Type:             "quiz",
		CorrectOptionIds: []int64{int64(correctAnswerIdx)},
		IsAnonymous:      &anonymous,
		Explanation:      "选择正确答案即可入群",
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
	a.scheduleVerifyTimeoutKick(b, cfg.ChatID, user.Id, timeout)
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
			CallbackData: CallbackData("verify", "check", strconv.FormatInt(cfg.ChatID, 10), strconv.FormatInt(user.Id, 10), strconv.Itoa(option)),
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
	a.scheduleVerifyTimeoutKick(b, cfg.ChatID, user.Id, timeout)
	return nil
}

func (a *App) scheduleVerifyTimeoutKick(b *gotgbot.Bot, chatID int64, userID int64, timeout time.Duration) {
	if timeout <= 0 {
		timeout = time.Minute
	}
	time.AfterFunc(timeout+5*time.Second, func() {
		if a.services.Redis == nil {
			return
		}
		key := fmt.Sprintf("unverified:%d:%d", chatID, userID)
		// Atomically delete the key. If Del returns 1 we are the ones removing it,
		// meaning the user never verified — safe to kick. If it returns 0 the key
		// was already gone (user passed verification), so we do nothing.
		// This eliminates the race between the Get→kick window and clearUnverifiedKey.
		n, err := a.services.Redis.Del(context.Background(), key).Result()
		if err == nil && n > 0 {
			_ = a.kickUnverifiedMember(b, chatID, userID)
		}
	})
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
	_ = a.postWelcomeMessage(b, ctx, chatID, cfg, *ctx.EffectiveUser)
	return nil
}

func (a *App) clearUnverifiedKey(chatID int64, userID int64) error {
	if a.services.Redis == nil {
		return nil
	}
	key := fmt.Sprintf("unverified:%d:%d", chatID, userID)
	return a.services.Redis.Del(context.Background(), key).Err()
}

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

func (a *App) handleAllowUser(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
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

// handleChatJoinRequest handles chat_join_request updates.
// When verify type is "turnstile", sends a private message with a Mini App WebApp button.
func (a *App) handleChatJoinRequest(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx == nil || ctx.ChatJoinRequest == nil || a.services.Admin == nil {
		return nil
	}
	chatID := ctx.ChatJoinRequest.Chat.Id
	userID := ctx.ChatJoinRequest.From.Id

	cfg, err := a.services.Admin.GetConfig(context.Background(), chatID)
	if err != nil || !cfg.VerifyEnabled || cfg.VerifyType != "turnstile" {
		return nil
	}

	link := a.generateTurnstileLink(chatID, userID)
	if link == "" {
		return nil
	}

	name := strings.TrimSpace(ctx.ChatJoinRequest.From.FirstName + " " + ctx.ChatJoinRequest.From.LastName)
	if name == "" {
		name = strconv.FormatInt(userID, 10)
	}

	text := fmt.Sprintf("👋 你好 %s！\n请点击下方按钮完成人机验证后即可入群。", name)
	_, err = b.SendMessageWithContext(context.Background(), userID, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: "🔒 完成验证", WebApp: &gotgbot.WebAppInfo{Url: link}}},
			},
		},
	})
	return err
}

// generateTurnstileLink builds a signed Mini App URL for Turnstile verification.
func (a *App) generateTurnstileLink(chatID, userID int64) string {
	if a.miniAppURL == "" || a.options.TurnstileVerifySecret == "" {
		return ""
	}
	exp := time.Now().Add(10 * time.Minute).Unix()
	sig := turnstileBotHMAC(a.options.TurnstileVerifySecret, chatID, userID, exp)
	return fmt.Sprintf("%s/#/verify?chat=%d&user=%d&sig=%s&exp=%d",
		strings.TrimRight(a.miniAppURL, "/"), chatID, userID, sig, exp)
}

// turnstileBotHMAC signs "chatID|userID|exp" using HMAC-SHA256.
// Must match the identical function in internal/api/verify_handler.go.
func turnstileBotHMAC(secret string, chatID, userID, exp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d|%d|%d", chatID, userID, exp)
	return hex.EncodeToString(mac.Sum(nil))
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
