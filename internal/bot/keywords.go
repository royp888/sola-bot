package bot

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const keywordMuteDuration = time.Hour
const spamMuteDuration = 10 * time.Minute
const moderationBlockedPointsKey = "moderation_blocked_points"

func (a *App) handleMessageModeration(b *gotgbot.Bot, ctx *ext.Context) error {
	if a.services.KeywordFilter == nil || ctx == nil || ctx.Message == nil {
		return ext.ContinueGroups
	}

	msg := ctx.Message
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		return ext.ContinueGroups
	}
	if msg.From == nil || msg.From.IsBot || len(msg.NewChatMembers) > 0 {
		return ext.ContinueGroups
	}
	if a.isMessageFromTelegramAdmin(b, ctx) {
		return ext.ContinueGroups
	}

	text := moderationMessageText(msg)
	scope := requestScope(ctx)

	// Restrict unverified users from sending links and media.
	if a.isUnverifiedUser(scope.Context, msg.Chat.Id, msg.From.Id) {
		modCfg, err := a.services.KeywordFilter.GetModerationConfig(scope.Context, msg.Chat.Id)
		if err == nil && modCfg.RestrictUnverified {
			if messageHasLink(msg, text) || messageHasMedia(msg) {
				_, _ = b.DeleteMessageWithContext(scope.Context, msg.Chat.Id, msg.MessageId, nil)
				_, _ = b.SendMessageWithContext(scope.Context, msg.From.Id, "请先完成验证，再发送链接或媒体。", nil)
				return ext.EndGroups
			}
		}
	}

	cfg, err := a.services.KeywordFilter.GetModerationConfig(scope.Context, msg.Chat.Id)
	if err != nil {
		return err
	}
	if cfg.SpamScoreThreshold <= 0 {
		cfg.SpamScoreThreshold = 60
	}

	match := KeywordFilterMatch{}
	if text != "" && cfg.KeywordFilterEnabled {
		match, err = a.services.KeywordFilter.MatchKeyword(scope.Context, msg.Chat.Id, text)
		if err != nil {
			return err
		}
	}

	score, reasons := a.spamScore(scope, msg, text, cfg, match)
	// AI secondary check for suspicious messages
	if score >= cfg.SpamScoreThreshold && score < 90 && !match.Matched && a.services.AiFilter != nil {
		userName := strings.TrimSpace(msg.From.FirstName + " " + msg.From.LastName)
		if userName == "" {
			userName = msg.From.Username
		}
		isSpam, reason, aiErr := a.services.AiFilter.IsSpam(scope.Context, text, userName)
		if aiErr == nil && isSpam {
			score = 90 // escalate to ban level
			reasons = append(reasons, "AI判定:"+reason)
		}
	}

	if match.Matched {
		action := normalizeBotKeywordAction(match.Action)
		duration := 0
		if action == "mute" {
			duration = int(keywordMuteDuration.Seconds())
		}
		if err := a.services.KeywordFilter.RecordKeywordViolation(scope.Context, KeywordViolation{
			UserID:          msg.From.Id,
			ChatID:          msg.Chat.Id,
			ViolationType:   "keyword_filter",
			ActionTaken:     action,
			MessageText:     truncateRunes(text, 1000),
			DetectedBy:      match.Keyword,
			DurationSeconds: duration,
		}); err != nil {
			return err
		}

		markModerationBlocked(ctx)
		if err := a.applyKeywordFilterAction(b, ctx, match, action); err != nil {
			return err
		}
		return ext.EndGroups
	}

	if score < cfg.SpamScoreThreshold {
		return ext.ContinueGroups
	}

	action, duration := spamAction(score, cfg.SpamScoreThreshold)
	if err := a.services.KeywordFilter.RecordKeywordViolation(scope.Context, KeywordViolation{
		UserID:          msg.From.Id,
		ChatID:          msg.Chat.Id,
		ViolationType:   "spam_score",
		ActionTaken:     action,
		MessageText:     truncateRunes(text, 1000),
		DetectedBy:      strings.Join(reasons, ","),
		DurationSeconds: duration,
	}); err != nil {
		return err
	}
	markModerationBlocked(ctx)
	if err := a.applySpamScoreAction(b, ctx, score, reasons, action, duration); err != nil {
		return err
	}
	return ext.EndGroups
}

func markModerationBlocked(ctx *ext.Context) {
	if ctx == nil {
		return
	}
	if ctx.Data == nil {
		ctx.Data = map[string]any{}
	}
	ctx.Data[moderationBlockedPointsKey] = true
}

func moderationBlockedPoints(ctx *ext.Context) bool {
	if ctx == nil || ctx.Data == nil {
		return false
	}
	blocked, _ := ctx.Data[moderationBlockedPointsKey].(bool)
	return blocked
}

func keywordPanelMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "刷新关键词", CallbackData: CallbackData("admin", "keywords")},
			{Text: "自动回复", CallbackData: CallbackData("admin", "autoreplies")},
		},
		{
			{Text: "消息模板", CallbackData: CallbackData("admin", "templates")},
			{Text: "返回群管", CallbackData: CallbackData("admin", "moderation")},
		},
		{
			{Text: "返回群组", CallbackData: CallbackData("menu", "group")},
		},
	}}}
}

func formatKeywordPanel(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		body = "当前还没有配置过滤关键词。"
	}
	return strings.Join([]string{
		"关键词过滤面板",
		"",
		"适合处理广告词、刷屏口令、引流词和高频违规语句。",
		body,
		"",
		"快捷命令",
		"/add_keyword 广告",
		"/del_keyword 广告",
	}, "\n")
}

func (a *App) handleKeywords(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.KeywordFilter == nil {
		return sendText(b, ctx, "关键词过滤服务尚未接入。", nil)
	}
	keywords, err := a.services.KeywordFilter.ListKeywords(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	keywords = strings.TrimSpace(keywords)
	if keywords == "" {
		keywords = "当前未配置过滤关键词。"
	}
	return respondText(b, ctx, formatKeywordPanel(keywords), keywordPanelMarkup())
}

func (a *App) handleAddKeyword(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.updateKeywordFilter(b, ctx, "add")
}

func (a *App) handleDelKeyword(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.updateKeywordFilter(b, ctx, "delete")
}

func (a *App) updateKeywordFilter(b *gotgbot.Bot, ctx *ext.Context, action string) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.KeywordFilter == nil {
		return sendText(b, ctx, "关键词过滤服务尚未接入。", nil)
	}

	keyword := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	if keyword == "" {
		if action == "add" {
			return sendText(b, ctx, "用法：/add_keyword 广告", nil)
		}
		return sendText(b, ctx, "用法：/del_keyword 广告", nil)
	}

	var (
		result string
		err    error
	)
	if action == "add" {
		result, err = a.services.KeywordFilter.AddKeyword(scope.Context, scope.Chat.ID, keyword, scope.Actor.ID)
	} else {
		result, err = a.services.KeywordFilter.DeleteKeyword(scope.Context, scope.Chat.ID, keyword, scope.Actor.ID)
	}
	if err != nil {
		return err
	}
	result = strings.TrimSpace(result)
	if result == "" {
		if action == "add" {
			result = "关键词已添加。"
		} else {
			result = "关键词已删除。"
		}
	}
	return respondText(b, ctx, result+"\n\n可继续增删关键词，或点按钮回到过滤面板。", keywordPanelMarkup())
}

func (a *App) auditKeywordAction(scope RequestScope, targetID int64, action string, detail string) {
	if a.services.AuditLog != nil {
		a.services.AuditLog.Log(AuditLogEntry{
			ActorTelegramID:  0,
			ChatTelegramID:   scope.Chat.ID,
			Action:           "keyword_filter",
			EntityType:       "user",
			TargetTelegramID: targetID,
			Detail:           fmt.Sprintf("action=%s %s", action, detail),
		})
	}
}

func (a *App) applyKeywordFilterAction(b *gotgbot.Bot, ctx *ext.Context, match KeywordFilterMatch, action string) error {
	scope := requestScope(ctx)
	msg := ctx.Message
	if msg == nil || msg.From == nil {
		return nil
	}

	reason := fmt.Sprintf("关键词命中：%s", match.Keyword)
	a.auditKeywordAction(scope, msg.From.Id, action, match.Keyword)
	_, deleteErr := b.DeleteMessageWithContext(scope.Context, msg.Chat.Id, msg.MessageId, nil)
	switch action {
	case "warn":
		if a.services.Admin == nil {
			return deleteErr
		}
		count, limit, err := a.services.Admin.RecordWarn(scope.Context, msg.Chat.Id, msg.From.Id, 0, reason)
		if err != nil {
			return err
		}
		if limit > 0 && int(count) >= limit {
			if _, err := b.BanChatMemberWithContext(scope.Context, msg.Chat.Id, msg.From.Id, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
				return err
			}
			_ = a.services.Admin.RecordBan(scope.Context, msg.Chat.Id, msg.From.Id, 0, fmt.Sprintf("关键词警告达到上限 %d", limit))
			return sendText(b, ctx, fmt.Sprintf("用户 %d 触发关键词过滤，警告 %d/%d，已自动封禁。", msg.From.Id, count, limit), nil)
		}
		return sendText(b, ctx, fmt.Sprintf("用户 %d 触发关键词过滤，已警告 %d/%d。", msg.From.Id, count, limit), nil)
	case "mute":
		until := time.Now().Add(keywordMuteDuration).Unix()
		if _, err := b.RestrictChatMemberWithContext(scope.Context, msg.Chat.Id, msg.From.Id, mutePermissions(), &gotgbot.RestrictChatMemberOpts{UntilDate: until, UseIndependentChatPermissions: true}); err != nil {
			return err
		}
		return sendText(b, ctx, fmt.Sprintf("用户 %d 触发关键词过滤，已禁言 %s。", msg.From.Id, keywordMuteDuration.Round(time.Second)), nil)
	case "ban":
		if _, err := b.BanChatMemberWithContext(scope.Context, msg.Chat.Id, msg.From.Id, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		if a.services.Admin != nil {
			_ = a.services.Admin.RecordBan(scope.Context, msg.Chat.Id, msg.From.Id, 0, reason)
		}
		return sendText(b, ctx, fmt.Sprintf("用户 %d 触发关键词过滤，已封禁。", msg.From.Id), nil)
	default:
		return deleteErr
	}
}

func (a *App) auditSpamAction(scope RequestScope, targetID int64, action string, score int, reasons []string) {
	if a.services.AuditLog != nil {
		a.services.AuditLog.Log(AuditLogEntry{
			ActorTelegramID:  0,
			ChatTelegramID:   scope.Chat.ID,
			Action:           "keyword_filter",
			EntityType:       "user",
			TargetTelegramID: targetID,
			Detail:           fmt.Sprintf("spam_score=%d action=%s reasons=%v", score, action, reasons),
		})
	}
}

func (a *App) applySpamScoreAction(b *gotgbot.Bot, ctx *ext.Context, score int, reasons []string, action string, durationSeconds int) error {
	scope := requestScope(ctx)
	msg := ctx.Message
	if msg == nil || msg.From == nil {
		return nil
	}
	reasonText := strings.Join(reasons, "、")
	a.auditSpamAction(scope, msg.From.Id, action, score, reasons)
	if reasonText == "" {
		reasonText = "规则命中"
	}
	_, deleteErr := b.DeleteMessageWithContext(scope.Context, msg.Chat.Id, msg.MessageId, nil)
	switch action {
	case "warn":
		if a.services.Admin == nil {
			return deleteErr
		}
		count, limit, err := a.services.Admin.RecordWarn(scope.Context, msg.Chat.Id, msg.From.Id, 0, fmt.Sprintf("spam_score=%d：%s", score, reasonText))
		if err != nil {
			return err
		}
		if limit > 0 && int(count) >= limit {
			if _, err := b.BanChatMemberWithContext(scope.Context, msg.Chat.Id, msg.From.Id, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
				return err
			}
			_ = a.services.Admin.RecordBan(scope.Context, msg.Chat.Id, msg.From.Id, 0, fmt.Sprintf("spam_score 警告达到上限 %d", limit))
			return sendText(b, ctx, fmt.Sprintf("用户 %d 疑似刷屏/广告，警告 %d/%d，已自动封禁。", msg.From.Id, count, limit), nil)
		}
		return sendText(b, ctx, fmt.Sprintf("用户 %d 疑似刷屏/广告，spam_score=%d，已警告 %d/%d。", msg.From.Id, score, count, limit), nil)
	case "mute":
		duration := time.Duration(durationSeconds) * time.Second
		if duration <= 0 {
			duration = spamMuteDuration
		}
		if _, err := b.RestrictChatMemberWithContext(scope.Context, msg.Chat.Id, msg.From.Id, mutePermissions(), &gotgbot.RestrictChatMemberOpts{UntilDate: time.Now().Add(duration).Unix(), UseIndependentChatPermissions: true}); err != nil {
			return err
		}
		return sendText(b, ctx, fmt.Sprintf("用户 %d 疑似刷屏/广告，spam_score=%d，已禁言 %s。", msg.From.Id, score, duration.Round(time.Second)), nil)
	case "ban":
		if _, err := b.BanChatMemberWithContext(scope.Context, msg.Chat.Id, msg.From.Id, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
			return err
		}
		if a.services.Admin != nil {
			_ = a.services.Admin.RecordBan(scope.Context, msg.Chat.Id, msg.From.Id, 0, fmt.Sprintf("spam_score=%d：%s", score, reasonText))
		}
		return sendText(b, ctx, fmt.Sprintf("用户 %d 疑似刷屏/广告，spam_score=%d，已封禁。", msg.From.Id, score), nil)
	default:
		return deleteErr
	}
}

func (a *App) spamScore(scope RequestScope, msg *gotgbot.Message, text string, cfg ChatModerationConfig, match KeywordFilterMatch) (int, []string) {
	score := 0
	reasons := []string{}
	mentionCount := countMentions(msg, text)
	if mentionCount > 3 {
		score += 30
		reasons = append(reasons, "多次提及")
	}
	if cfg.BlockLinks && messageHasBlockedLink(msg, text, cfg.LinkWhitelist, cfg.LinkBlacklist) {
		score += 25
		reasons = append(reasons, "链接")
	}
	if cfg.BlockForwards && msg.ForwardOrigin != nil {
		score += 20
		reasons = append(reasons, "转发")
	}
	if cfg.BlockMedia && messageHasMedia(msg) {
		score += 25
		reasons = append(reasons, "媒体")
	}
	if isShortEmojiLike(text) {
		score += 15
		reasons = append(reasons, "短表情消息")
	}
	if a.services.RateLimit != nil && scope.Actor.ID != 0 {
		allowed, _, err := a.services.RateLimit.Allow(scope.Context, fmt.Sprintf("spam:activity:%d:%d", scope.Chat.ID, scope.Actor.ID), 1)
		if err == nil && !allowed {
			score += 10
			reasons = append(reasons, "高频发言")
		}
	}
	if match.Matched {
		score += 20
		reasons = append(reasons, "关键词")
	}
	return score, reasons
}

func (a *App) isMessageFromTelegramAdmin(b *gotgbot.Bot, ctx *ext.Context) bool {
	scope := requestScope(ctx)
	if a.services.TelegramAccess == nil || scope.Chat.ID == 0 || scope.Actor.ID == 0 {
		return false
	}
	status, err := a.services.TelegramAccess.CheckUserAdmin(scope.Context, b, scope.Chat.ID, scope.Actor.ID)
	return err == nil && (status.IsAdmin || status.Status == "creator")
}

func spamAction(score int, threshold int) (string, int) {
	switch {
	case score >= 90:
		return "ban", 0
	case score >= 75:
		return "mute", int(spamMuteDuration.Seconds())
	case score >= threshold:
		return "warn", 0
	default:
		return "delete_message", 0
	}
}

func countMentions(msg *gotgbot.Message, text string) int {
	count := strings.Count(text, "@")
	for _, entity := range append(msg.Entities, msg.CaptionEntities...) {
		switch entity.Type {
		case "mention", "text_mention":
			count++
		}
	}
	return count
}

func messageHasLink(msg *gotgbot.Message, text string) bool {
	for _, entity := range append(msg.Entities, msg.CaptionEntities...) {
		switch entity.Type {
		case "url", "text_link", "email":
			return true
		}
	}
	return regexp.MustCompile(`(?i)\b((https?://|t\.me/|telegram\.me/|www\.)\S+)`).MatchString(text)
}

func messageHasBlockedLink(msg *gotgbot.Message, text string, whitelist, blacklist []string) bool {
	links := extractLinks(msg, text)
	if len(links) == 0 {
		return false
	}
	// Whitelist takes priority: if whitelist is set, only whitelisted domains pass.
	if len(whitelist) > 0 {
		for _, link := range links {
			host := extractHost(link)
			if host != "" && !domainMatches(host, whitelist) {
				return true
			}
		}
		return false
	}
	// Blacklist mode: if blacklist is set, block matching domains.
	if len(blacklist) > 0 {
		for _, link := range links {
			host := extractHost(link)
			if host != "" && domainMatches(host, blacklist) {
				return true
			}
		}
		return false
	}
	// No whitelist or blacklist: block all links (original behaviour).
	return true
}

func extractLinks(msg *gotgbot.Message, text string) []string {
	seen := map[string]bool{}
	var links []string
	for _, entity := range append(msg.Entities, msg.CaptionEntities...) {
		switch entity.Type {
		case "url":
			u := entityText(msg, entity)
			u = strings.TrimSpace(u)
			if u != "" && !seen[u] {
				seen[u] = true
				links = append(links, u)
			}
		case "text_link":
			u := strings.TrimSpace(entity.Url)
			if u != "" && !seen[u] {
				seen[u] = true
				links = append(links, u)
			}
		}
	}
	re := regexp.MustCompile(`(?i)\b((https?://|t\.me/|telegram\.me/|www\.)\S+)`)
	for _, m := range re.FindAllString(text, -1) {
		m = strings.TrimSpace(m)
		if m != "" && !seen[m] {
			seen[m] = true
			links = append(links, m)
		}
	}
	return links
}

func extractHost(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	return host
}

func domainMatches(host string, domains []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if host == d {
			return true
		}
		if strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func entityText(msg *gotgbot.Message, entity gotgbot.MessageEntity) string {
	if msg == nil {
		return ""
	}
	text := msg.Text
	if entity.Offset >= int64(len(text)) && msg.Caption != "" {
		text = msg.Caption
	}
	if entity.Offset >= int64(len(text)) {
		return ""
	}
	end := entity.Offset + entity.Length
	if end > int64(len(text)) {
		end = int64(len(text))
	}
	return text[entity.Offset:end]
}

func messageHasMedia(msg *gotgbot.Message) bool {
	return msg.Photo != nil || msg.Video != nil || msg.Document != nil || msg.Sticker != nil || msg.Animation != nil || msg.Audio != nil || msg.Voice != nil || msg.VideoNote != nil
}

func isShortEmojiLike(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	runes := []rune(text)
	if len(runes) > 4 {
		return false
	}
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func moderationMessageText(msg *gotgbot.Message) string {
	if msg == nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimSpace(msg.Text) + "\n" + strings.TrimSpace(msg.Caption))
}

func normalizeBotKeywordAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "warn", "mute", "ban":
		return strings.ToLower(strings.TrimSpace(action))
	default:
		return "delete_message"
	}
}

func truncateRunes(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}

// isUnverifiedUser returns true if the user has an active unverified key in Redis.
func (a *App) isUnverifiedUser(ctx context.Context, chatID int64, userID int64) bool {
	if a.services.Redis == nil {
		return false
	}
	key := fmt.Sprintf("unverified:%d:%d", chatID, userID)
	_, err := a.services.Redis.Get(ctx, key).Result()
	return err == nil
}
