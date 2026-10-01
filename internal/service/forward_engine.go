package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

const (
	forwardRuleCacheTTL    = 15 * time.Second
	forwardMaxMediaCaption = 1024
	forwardMaxTextLength   = 4096
	// Telegram 服务端复制（copyMessage）单文件上限 2GB，不需要机器人下载再上传。
	forwardMaxCopyBytes      = int64(2) * 1000 * 1000 * 1000
	forwardDefaultProductIDs = `(?i)(?:#|＃|ID[:：]?\s*)(\d{2,12})`
	// 相册合并：攒够窗口期再 copyMessages
	forwardAlbumWindow = 1500 * time.Millisecond
	forwardAlbumMax    = 10
)

var (
	forwardURLRegex        = regexp.MustCompile(`(?i)\b(?:https?://|www\.|t\.me/)\S+`)
	forwardMentionRegex    = regexp.MustCompile(`@[A-Za-z0-9_]{4,32}`)
	forwardHashtagRegex    = regexp.MustCompile(`[#＃][\p{L}\p{N}_]{1,32}`)
	forwardSourceLineRegex = regexp.MustCompile(`(?i)(来源|來自|来自|转发自|轉發自|搬运|搬運|via\b|source\s*[:：])`)
	forwardRetryAfterRegex = regexp.MustCompile(`(?i)retry after (\d+)`)
	forwardBlankLineRegex  = regexp.MustCompile(`\n{3,}`)
)

// ForwardEngine 是转发功能的核心：实时监听 + 历史批量都走这里。
// 同时被 bot 进程（实时）与 worker 进程（历史）复用。
type ForwardEngine struct {
	store *store.Store
	log   *zap.Logger
	token string

	mu       sync.Mutex
	cachedAt time.Time
	cached   []model.AutoForward

	albumMu sync.Mutex
	albums  map[string]*forwardAlbumGroup

	botMu sync.Mutex
	bot   *gotgbot.Bot
}

// forwardAlbumGroup 是相册合并的临时缓冲。
type forwardAlbumGroup struct {
	rule       model.AutoForward
	chatID     int64
	messageIDs []int64
	timer      *time.Timer
}

// NewForwardEngine 创建转发引擎。logger 可为 nil；token 仅在 API 侧“预览/试发”时需要。
func NewForwardEngine(st *store.Store, token string, logger *zap.Logger) *ForwardEngine {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ForwardEngine{store: st, token: strings.TrimSpace(token), log: logger}
}

// Client 返回用于预览/试发的机器人客户端（懒加载）。
func (e *ForwardEngine) Client() (*gotgbot.Bot, error) {
	if e == nil {
		return nil, errors.New("forward engine is not configured")
	}
	e.botMu.Lock()
	defer e.botMu.Unlock()
	if e.bot != nil {
		return e.bot, nil
	}
	if e.token == "" {
		return nil, errors.New("bot token is not configured")
	}
	bot, err := gotgbot.NewBot(e.token, nil)
	if err != nil {
		return nil, err
	}
	e.bot = bot
	return bot, nil
}

// ---------------------------------------------------------------- 实时监听

// HandleRealtime 由 bot 进程在收到新消息（含频道 post）时调用。
// 只处理配置为 realtime 的规则；任何错误都不应影响机器人其它功能。
func (e *ForwardEngine) HandleRealtime(ctx context.Context, b *gotgbot.Bot, msg *gotgbot.Message) error {
	if e == nil || e.store == nil || e.store.DB == nil || b == nil || msg == nil {
		return nil
	}
	sourceChatID := msg.Chat.Id
	if sourceChatID == 0 {
		return nil
	}
	// 防止机器人自己发到源聊天的消息被再次转发（死循环）
	if msg.From != nil && msg.From.Id == b.Id {
		return nil
	}
	if msg.Chat.Type == "private" {
		return nil
	}

	rules, err := e.realtimeRules(ctx, sourceChatID)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		if reason := shouldSkipForward(rule, b, msg); reason != "" {
			e.recordSkipped(ctx, rule, sourceChatID, msg.MessageId, reason, msg)
			continue
		}
		if rule.AlbumBatch && e.bufferAlbum(b, rule, sourceChatID, msg) {
			continue
		}
		e.forwardMessage(ctx, b, rule, sourceChatID, msg.MessageId, msg)
	}
	return nil
}

// bufferAlbum 把相册（media group）的成员消息攒够窗口期再一次 copyMessages 过去，
// 这样目标频道里仍然是一组相册。返回 true 表示已进入缓冲、不需要单条处理。
// 代价：相册那一组不能改文案、不能挂按钮（Telegram 的 sendMediaGroup 不支持按钮）。
func (e *ForwardEngine) bufferAlbum(b *gotgbot.Bot, rule model.AutoForward, sourceChatID int64, msg *gotgbot.Message) bool {
	groupID := strings.TrimSpace(msg.MediaGroupId)
	if groupID == "" {
		return false
	}
	key := fmt.Sprintf("%d:%d:%s", rule.ID, sourceChatID, groupID)

	e.albumMu.Lock()
	if e.albums == nil {
		e.albums = make(map[string]*forwardAlbumGroup)
	}
	group, ok := e.albums[key]
	if !ok {
		group = &forwardAlbumGroup{rule: rule, chatID: sourceChatID}
		e.albums[key] = group
		group.timer = time.AfterFunc(forwardAlbumWindow, func() { e.flushAlbum(b, key) })
	}
	group.messageIDs = append(group.messageIDs, msg.MessageId)
	full := len(group.messageIDs) >= forwardAlbumMax
	e.albumMu.Unlock()

	if full {
		e.flushAlbum(b, key)
	}
	return true
}

func (e *ForwardEngine) flushAlbum(b *gotgbot.Bot, key string) {
	e.albumMu.Lock()
	group := e.albums[key]
	delete(e.albums, key)
	e.albumMu.Unlock()
	if group == nil || len(group.messageIDs) == 0 {
		return
	}
	if group.timer != nil {
		group.timer.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ids := group.messageIDs
	if len(ids) == 1 {
		e.forwardMessageWithRetry(ctx, b, group.rule, group.chatID, ids[0], nil)
		return
	}
	if _, err := b.CopyMessagesWithContext(ctx, group.rule.TargetChatID, group.chatID, ids, nil); err == nil {
		for _, messageID := range ids {
			e.recordResult(ctx, group.rule, group.chatID, messageID, forwardResult{
				Status:       model.ForwardLogSent,
				TargetChatID: group.rule.TargetChatID,
			})
		}
		return
	} else {
		e.log.Warn("album copy failed, falling back to single copies",
			zap.Uint64("rule_id", group.rule.ID), zap.Error(err))
	}
	// copyMessages 是原子的：整体失败（例如相册里有一条已删除）时退回逐条复制，
	// 逐条走 copyMessage，反而可以套用文案与购买按钮。
	for _, messageID := range ids {
		e.forwardMessageWithRetry(ctx, b, group.rule, group.chatID, messageID, nil)
	}
}

func (e *ForwardEngine) realtimeRules(ctx context.Context, sourceChatID int64) ([]model.AutoForward, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if time.Since(e.cachedAt) > forwardRuleCacheTTL || e.cached == nil {
		var rules []model.AutoForward
		err := e.store.DB.WithContext(ctx).
			Where("enabled = ? AND realtime = ?", true, true).
			Find(&rules).Error
		if err != nil {
			if isMissingTableError(err) {
				return nil, nil
			}
			return nil, err
		}
		e.cached = rules
		e.cachedAt = time.Now()
	}
	out := make([]model.AutoForward, 0, 2)
	for _, rule := range e.cached {
		if rule.SourceChatID == sourceChatID {
			out = append(out, rule)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------- 历史批量

// ProcessHistoryBatch 推进一条规则的历史补发，返回本轮处理的条数。
func (e *ForwardEngine) ProcessHistoryBatch(ctx context.Context, b *gotgbot.Bot, rule model.AutoForward) (int, error) {
	if e == nil || e.store == nil || e.store.DB == nil || b == nil {
		return 0, nil
	}
	if !rule.Enabled || rule.HistoryState != model.ForwardStateRunning {
		return 0, nil
	}
	if rule.SourceChatID == 0 || rule.TargetChatID == 0 {
		return 0, e.haltHistory(ctx, rule, "源/目标频道未配置")
	}

	cursor := rule.HistoryCursor
	if cursor <= 0 {
		cursor = rule.HistoryFromID
	}
	if cursor <= 0 {
		return 0, e.haltHistory(ctx, rule, "历史起始消息 ID 未配置")
	}

	batch := rule.BatchSize
	if batch <= 0 {
		batch = 20
	}
	delay := time.Duration(rule.DelayMS) * time.Millisecond
	if delay <= 0 {
		delay = 500 * time.Millisecond
	}
	missLimit := rule.HistoryAutoStopMisses
	if missLimit <= 0 {
		missLimit = 25
	}

	processed := 0
	missStreak := 0
	for i := 0; i < batch; i++ {
		if ctx.Err() != nil {
			break
		}
		if rule.HistoryToID > 0 && cursor > rule.HistoryToID {
			return processed, e.finishHistory(ctx, rule)
		}
		messageID := cursor
		cursor++

		result := e.forwardMessageWithRetry(ctx, b, rule, rule.SourceChatID, messageID, nil)
		processed++

		switch {
		case result.Sent():
			missStreak = 0
		case result.Status == model.ForwardLogSkipped && result.SkipReason == forwardSkipNotFound:
			missStreak++
		default:
			missStreak = 0
		}

		if err := e.updateHistoryProgress(ctx, rule, messageID, result); err != nil {
			e.log.Warn("update forward history progress failed", zap.Uint64("rule_id", rule.ID), zap.Error(err))
		}

		// 未指定结束 ID 时：连续多条“消息不存在”即认为已经追上频道尾部
		if rule.HistoryToID <= 0 && missStreak >= missLimit {
			e.log.Info("forward history reached channel tail",
				zap.Uint64("rule_id", rule.ID), zap.Int64("cursor", messageID), zap.Int("misses", missStreak))
			return processed, e.finishHistory(ctx, rule)
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		if result.Status == model.ForwardLogFailed && result.Err != nil && isMessageNotFound(result.Err) {
			missStreak++
		}
	}
	return processed, nil
}

const forwardSkipNotFound = "message_not_found"

func (e *ForwardEngine) haltHistory(ctx context.Context, rule model.AutoForward, reason string) error {
	return e.store.DB.WithContext(ctx).Model(&model.AutoForward{}).
		Where("id = ?", rule.ID).
		Updates(map[string]any{
			"history_state": model.ForwardStateFailed,
			"last_error":    reason,
			"updated_at":    time.Now(),
		}).Error
}

func (e *ForwardEngine) finishHistory(ctx context.Context, rule model.AutoForward) error {
	now := time.Now()
	return e.store.DB.WithContext(ctx).Model(&model.AutoForward{}).
		Where("id = ?", rule.ID).
		Updates(map[string]any{
			"history_state":       model.ForwardStateCompleted,
			"history_finished_at": now,
			"updated_at":          now,
		}).Error
}

func (e *ForwardEngine) updateHistoryProgress(ctx context.Context, rule model.AutoForward, processedID int64, result forwardResult) error {
	updates := map[string]any{
		"history_cursor": processedID,
		"updated_at":     time.Now(),
	}
	switch result.Status {
	case model.ForwardLogSent:
		updates["history_sent"] = gorm.Expr("history_sent + 1")
	case model.ForwardLogFailed:
		updates["history_failed"] = gorm.Expr("history_failed + 1")
	default:
		updates["history_skipped"] = gorm.Expr("history_skipped + 1")
	}
	return e.store.DB.WithContext(ctx).Model(&model.AutoForward{}).Where("id = ?", rule.ID).Updates(updates).Error
}

// bumpRuleCounters 累加规则维度的统计（实时与历史共用，每次记录日志时各加一次）。
func (e *ForwardEngine) bumpRuleCounters(ctx context.Context, rule model.AutoForward, result forwardResult) {
	updates := map[string]any{"updated_at": time.Now()}
	switch result.Status {
	case model.ForwardLogSent:
		updates["total_forwarded"] = gorm.Expr("total_forwarded + 1")
		updates["last_forward_at"] = time.Now()
	case model.ForwardLogFailed:
		updates["total_failed"] = gorm.Expr("total_failed + 1")
		if result.Err != nil {
			updates["last_error"] = truncateRunes(result.Err.Error(), 500)
		}
	default:
		updates["total_skipped"] = gorm.Expr("total_skipped + 1")
	}
	err := e.store.DB.WithContext(ctx).Model(&model.AutoForward{}).Where("id = ?", rule.ID).Updates(updates).Error
	if err != nil && !isMissingTableError(err) {
		e.log.Warn("bump forward counters failed", zap.Uint64("rule_id", rule.ID), zap.Error(err))
	}
}

// ---------------------------------------------------------------- 单条转发

type forwardResult struct {
	Status          string
	SkipReason      string
	TargetChatID    int64
	TargetMessageID int64
	MediaType       string
	FileSize        int64
	Text            string
	Err             error
}

func (r forwardResult) Sent() bool    { return r.Status == model.ForwardLogSent }
func (r forwardResult) Skipped() bool { return r.Status == model.ForwardLogSkipped }

// forwardMessageWithRetry 在 FloodWait 时自动等待并重试一次。
func (e *ForwardEngine) forwardMessageWithRetry(ctx context.Context, b *gotgbot.Bot, rule model.AutoForward, sourceChatID, messageID int64, msg *gotgbot.Message) forwardResult {
	result := e.forwardMessage(ctx, b, rule, sourceChatID, messageID, msg)
	if result.Err == nil {
		return result
	}
	wait := retryAfterSeconds(result.Err)
	if wait <= 0 || wait > 15*60 {
		return result
	}
	e.log.Info("forward flood wait", zap.Uint64("rule_id", rule.ID), zap.Int64("seconds", wait))
	select {
	case <-ctx.Done():
		return result
	case <-time.After(time.Duration(wait) * time.Second):
	}
	return e.forwardMessage(ctx, b, rule, sourceChatID, messageID, msg)
}

func (e *ForwardEngine) forwardMessage(ctx context.Context, b *gotgbot.Bot, rule model.AutoForward, sourceChatID, messageID int64, msg *gotgbot.Message) forwardResult {
	result := forwardResult{Status: model.ForwardLogFailed, TargetChatID: rule.TargetChatID}
	if msg != nil {
		result.MediaType = forwardMediaType(msg)
		result.FileSize = forwardMediaSize(msg)
		result.Text = forwardMessageText(msg)
	}

	if rule.TargetChatID == 0 {
		result.Err = errors.New("目标频道未配置")
		return result
	}
	if messageID <= 0 {
		result.Err = errors.New("源消息 ID 无效")
		return result
	}
	// 去重：历史补发与实时监听可能命中同一条消息
	if rule.SkipDuplicates && e.alreadyForwarded(ctx, rule.ID, sourceChatID, messageID) {
		result.Status = model.ForwardLogSkipped
		result.SkipReason = "duplicate"
		return result
	}

	rendered := renderForward(rule, result.Text, messageID, sourceChatID)
	result.Text = truncateRunes(rendered.caption, 300)

	isText := result.MediaType == ""
	keyboard, kbErr := buildForwardKeyboard(rule, rendered.buyURL, isText)
	if kbErr != nil {
		e.log.Warn("forward inline keyboard invalid", zap.Uint64("rule_id", rule.ID), zap.Error(kbErr))
	}

	// 没有改写需求时完全按原样复制，保留原始 entities/排版，也避免 HTML 解析报错
	if !rendered.changed && rule.Mode != model.ForwardModeForward {
		opts := &gotgbot.CopyMessageOpts{}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, err := b.CopyMessageWithContext(ctx, rule.TargetChatID, sourceChatID, messageID, opts)
		if err != nil {
			if isMessageNotFound(err) {
				result.Status = model.ForwardLogSkipped
				result.SkipReason = forwardSkipNotFound
				result.Err = err
				return result
			}
			result.Err = err
			e.recordResult(ctx, rule, sourceChatID, messageID, result)
			return result
		}
		result.Status = model.ForwardLogSent
		result.TargetMessageID = sent.MessageId
		e.recordResult(ctx, rule, sourceChatID, messageID, result)
		return result
	}

	var (
		targetMessageID int64
		sendErr         error
	)
	switch {
	case rule.Mode == model.ForwardModeForward:
		sent, err := b.ForwardMessageWithContext(ctx, rule.TargetChatID, sourceChatID, messageID, nil)
		if err != nil {
			sendErr = err
		} else {
			targetMessageID = sent.MessageId
		}
	case isText && rendered.changed && msg != nil:
		// 纯文本消息的文案只能通过重新发送来改写（copyMessage 的 caption 对文本无效）
		text := truncateRunes(rendered.caption, forwardMaxTextLength)
		if strings.TrimSpace(text) == "" {
			result.Status = model.ForwardLogSkipped
			result.SkipReason = "empty_caption"
			return result
		}
		opts := &gotgbot.SendMessageOpts{}
		if rule.ParseMode != "" {
			opts.ParseMode = rule.ParseMode
		}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, err := b.SendMessageWithContext(ctx, rule.TargetChatID, text, opts)
		if err != nil {
			sendErr = err
		} else {
			targetMessageID = sent.MessageId
		}
	default:
		// copyMessage 在 Telegram 服务端搬运，媒体不需要经机器人下载，
		// 因此 2GB 以内的大文件也能原样转发，并且可以同时改文案 + 挂按钮。
		if result.FileSize > forwardMaxCopyBytes {
			result.Status = model.ForwardLogSkipped
			result.SkipReason = "file_too_large"
			result.Err = fmt.Errorf("文件 %.1fGB 超出 2GB 上限", float64(result.FileSize)/1e9)
			return result
		}
		opts := &gotgbot.CopyMessageOpts{}
		if rule.ParseMode != "" {
			opts.ParseMode = rule.ParseMode
		}
		caption := truncateRunes(rendered.caption, forwardMaxMediaCaption)
		if strings.TrimSpace(caption) != "" {
			opts.Caption = &caption
		}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, err := b.CopyMessageWithContext(ctx, rule.TargetChatID, sourceChatID, messageID, opts)
		if err != nil {
			sendErr = err
		} else {
			targetMessageID = sent.MessageId
		}
	}

	if sendErr != nil {
		if isMessageNotFound(sendErr) {
			result.Status = model.ForwardLogSkipped
			result.SkipReason = forwardSkipNotFound
			result.Err = sendErr
			if msg != nil {
				e.recordSkipped(ctx, rule, sourceChatID, messageID, forwardSkipNotFound, msg)
			}
			return result
		}
		result.Err = sendErr
		e.recordResult(ctx, rule, sourceChatID, messageID, result)
		return result
	}

	result.Status = model.ForwardLogSent
	result.TargetMessageID = targetMessageID
	if !rendered.changed {
		result.Text = truncateRunes(rendered.caption, 300)
	}
	e.recordResult(ctx, rule, sourceChatID, messageID, result)
	return result
}

func (e *ForwardEngine) alreadyForwarded(ctx context.Context, ruleID uint64, sourceChatID, messageID int64) bool {
	var count int64
	err := e.store.DB.WithContext(ctx).Model(&model.AutoForwardLog{}).
		Where("forward_id = ? AND source_chat_id = ? AND source_message_id = ? AND status = ?",
			ruleID, sourceChatID, messageID, model.ForwardLogSent).
		Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}

func (e *ForwardEngine) recordSkipped(ctx context.Context, rule model.AutoForward, sourceChatID, messageID int64, reason string, msg *gotgbot.Message) {
	result := forwardResult{
		Status:       model.ForwardLogSkipped,
		SkipReason:   reason,
		TargetChatID: rule.TargetChatID,
	}
	if msg != nil {
		result.MediaType = forwardMediaType(msg)
		result.FileSize = forwardMediaSize(msg)
		result.Text = truncateRunes(forwardMessageText(msg), 300)
	}
	e.recordResult(ctx, rule, sourceChatID, messageID, result)
}

func (e *ForwardEngine) recordResult(ctx context.Context, rule model.AutoForward, sourceChatID, messageID int64, result forwardResult) {
	entry := model.AutoForwardLog{
		ForwardID:       rule.ID,
		RuleName:        rule.Name,
		SourceChatID:    sourceChatID,
		SourceMessageID: messageID,
		TargetChatID:    result.TargetChatID,
		TargetMessageID: result.TargetMessageID,
		Status:          result.Status,
		MediaType:       result.MediaType,
		FileSize:        result.FileSize,
		Text:            result.Text,
		CreatedAt:       time.Now(),
	}
	if result.Err != nil {
		entry.Error = truncateRunes(result.Err.Error(), 500)
	}
	if entry.Status == "" {
		entry.Status = model.ForwardLogSent
	}
	err := e.store.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "forward_id"}, {Name: "source_chat_id"}, {Name: "source_message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "target_chat_id", "target_message_id", "error", "text", "media_type", "file_size", "created_at"}),
	}).Create(&entry).Error
	if err != nil && !isMissingTableError(err) {
		e.log.Warn("save forward log failed", zap.Uint64("rule_id", rule.ID), zap.Error(err))
	}
	e.bumpRuleCounters(ctx, rule, result)
}

// TestSend 把当前规则的文案 + 按钮发到目标频道，用于上线前自检。
func (e *ForwardEngine) TestSend(ctx context.Context, rule model.AutoForward, req api.AutoForwardPreviewRequest) (*model.AutoForwardLog, error) {
	bot, err := e.Client()
	if err != nil {
		return nil, err
	}
	if rule.TargetChatID == 0 {
		return nil, errors.New("目标频道未配置")
	}
	preview := renderForward(rule, req.Text, req.MessageID, rule.SourceChatID)
	text := truncateRunes(preview.caption, forwardMaxTextLength)
	if strings.TrimSpace(text) == "" {
		text = "（测试消息：请在规则里填写文案模板）"
	}
	keyboard, kbErr := buildForwardKeyboard(rule, preview.buyURL, true)
	if kbErr != nil {
		return nil, kbErr
	}
	opts := &gotgbot.SendMessageOpts{}
	if rule.ParseMode != "" {
		opts.ParseMode = rule.ParseMode
	}
	if keyboard != nil {
		opts.ReplyMarkup = keyboard
	}
	sent, err := bot.SendMessageWithContext(ctx, rule.TargetChatID, text, opts)
	if err != nil {
		return nil, err
	}

	entry := model.AutoForwardLog{
		ForwardID:       rule.ID,
		RuleName:        rule.Name,
		SourceChatID:    rule.SourceChatID,
		SourceMessageID: -time.Now().UnixNano(),
		TargetChatID:    rule.TargetChatID,
		TargetMessageID: sent.MessageId,
		Status:          model.ForwardLogSent,
		MediaType:       "test",
		Text:            truncateRunes(text, 300),
		CreatedAt:       time.Now(),
	}
	_ = e.store.DB.WithContext(ctx).Create(&entry).Error
	return &entry, nil
}

// RenderPreview 在界面上预览最终文案与按钮。
func RenderForwardPreview(rule model.AutoForward, req api.AutoForwardPreviewRequest) api.AutoForwardPreviewResponse {
	rendered := renderForward(rule, req.Text, req.MessageID, rule.SourceChatID)
	isText := strings.TrimSpace(req.MediaType) == "" || req.MediaType == "text"
	keyboard, err := buildForwardKeyboard(rule, rendered.buyURL, isText)
	resp := api.AutoForwardPreviewResponse{
		Caption:       rendered.caption,
		BuyURL:        rendered.buyURL,
		BuyButtonText: rule.BuyText,
		Warnings:      append([]string{}, rendered.warnings...),
	}
	if keyboard != nil {
		for _, row := range keyboard.InlineKeyboard {
			out := api.InlineKeyboardRow{}
			for _, button := range row {
				out.Buttons = append(out.Buttons, api.InlineKeyboardButton{Text: button.Text, URL: button.Url})
			}
			resp.Buttons = append(resp.Buttons, out)
		}
	}
	if err != nil {
		resp.Warnings = append(resp.Warnings, err.Error())
	}
	if reason := shouldSkipForwardPreview(rule, req); reason != "" {
		resp.Skipped = true
		resp.SkipReason = reason
	}
	return resp
}

// ---------------------------------------------------------------- 文案渲染

type renderedForward struct {
	caption  string
	buyURL   string
	changed  bool
	warnings []string
}

func renderForward(rule model.AutoForward, sourceText string, messageID, sourceChatID int64) renderedForward {
	text := sanitizeForwardText(sourceText)

	for _, pair := range parseForwardReplacements(rule.ReplacementsJSON) {
		if pair.From == "" {
			continue
		}
		text = strings.ReplaceAll(text, pair.From, pair.To)
	}
	if rule.DropSourceLine {
		text = dropSourceLines(text)
	}
	if rule.RemoveLinks {
		text = forwardURLRegex.ReplaceAllString(text, "")
	}
	if rule.RemoveMentions {
		text = forwardMentionRegex.ReplaceAllString(text, "")
	}
	if rule.RemoveHashtags {
		text = forwardHashtagRegex.ReplaceAllString(text, "")
	}
	text = forwardBlankLineRegex.ReplaceAllString(collapseSpacesPerLine(text), "\n\n")
	text = strings.TrimSpace(text)

	buyURL := renderBuyURL(rule, sourceText, messageID, sourceChatID)
	out := renderedForward{
		buyURL:  buyURL,
		changed: forwardRewriteRequested(rule),
	}

	insert := text
	title := forwardTitle(text)
	if strings.EqualFold(strings.TrimSpace(rule.ParseMode), "HTML") {
		insert = html.EscapeString(text)
		title = html.EscapeString(title)
	}
	values := map[string]string{
		"{text}":      insert,
		"{title}":     title,
		"{source}":    rule.SourceTitle,
		"{source_id}": strconv.FormatInt(rule.SourceChatID, 10),
		"{chat_id}":   strconv.FormatInt(sourceChatID, 10),
		"{msg_id}":    strconv.FormatInt(messageID, 10),
		"{link}":      telegramMessageLink(sourceChatID, messageID),
		"{buy_url}":   buyURL,
		"{date}":      time.Now().Format("2006-01-02"),
	}

	if template := strings.TrimSpace(rule.CaptionTemplate); template != "" {
		out.caption = template
		for key, value := range values {
			out.caption = strings.ReplaceAll(out.caption, key, value)
		}
		if !strings.Contains(template, "{text}") {
			// 模板里没写 {text}：按开关决定是否保留原文
			if rule.KeepOriginalText && text != "" {
				out.caption = strings.TrimSpace(out.caption) + "\n\n" + insert
			}
			if strings.TrimSpace(out.caption) == "" && !rule.KeepOriginalText {
				out.warnings = append(out.warnings, "文案模板没有 {text}，且已关闭“保留原文”，转发消息将没有文字")
			}
		}
	} else {
		parts := make([]string, 0, 3)
		if prefix := strings.TrimSpace(rule.CaptionPrefix); prefix != "" {
			parts = append(parts, prefix)
		}
		if text != "" {
			parts = append(parts, insert)
		}
		if suffix := strings.TrimSpace(rule.CaptionSuffix); suffix != "" {
			parts = append(parts, suffix)
		}
		out.caption = strings.TrimSpace(strings.Join(parts, "\n\n"))
	}

	if rule.BuyEnabled {
		switch {
		case strings.TrimSpace(rule.BuyURLTemplate) == "":
			out.warnings = append(out.warnings, "已开启购买按钮但没有填写购买链接模板，按钮不会出现")
		case buyURL == "":
			out.warnings = append(out.warnings, "购买链接模板里的 {product_id} 在原文中没匹配到，按钮不会出现")
		}
	}
	if len([]rune(out.caption)) > forwardMaxMediaCaption {
		out.warnings = append(out.warnings, "文案超过 1024 字，转发媒体消息时会被截断")
	}
	return out
}

func forwardRewriteRequested(rule model.AutoForward) bool {
	return strings.TrimSpace(rule.CaptionTemplate) != "" ||
		strings.TrimSpace(rule.CaptionPrefix) != "" ||
		strings.TrimSpace(rule.CaptionSuffix) != "" ||
		rule.RemoveLinks || rule.RemoveMentions || rule.RemoveHashtags || rule.DropSourceLine ||
		strings.TrimSpace(rule.ReplacementsJSON) != ""
}

type forwardReplacement struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func parseForwardReplacements(raw string) []forwardReplacement {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []forwardReplacement
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func renderBuyURL(rule model.AutoForward, sourceText string, messageID, sourceChatID int64) string {
	if !rule.BuyEnabled {
		return ""
	}
	template := strings.TrimSpace(rule.BuyURLTemplate)
	if template == "" {
		return ""
	}
	productID := extractProductID(rule, sourceText)
	if strings.Contains(template, "{product_id}") && productID == "" {
		return ""
	}
	url := template
	url = strings.ReplaceAll(url, "{product_id}", productID)
	url = strings.ReplaceAll(url, "{msg_id}", strconv.FormatInt(messageID, 10))
	url = strings.ReplaceAll(url, "{chat_id}", strconv.FormatInt(sourceChatID, 10))
	url = strings.ReplaceAll(url, "{slug}", slugifyForwardText(sourceText))
	url = strings.ReplaceAll(url, "{date}", time.Now().Format("20060102"))
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return ""
	}
	return url
}

func extractProductID(rule model.AutoForward, text string) string {
	pattern := strings.TrimSpace(rule.ProductIDRegex)
	if pattern == "" {
		pattern = forwardDefaultProductIDs
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	match := re.FindStringSubmatch(text)
	if len(match) == 0 {
		return ""
	}
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return strings.TrimSpace(match[0])
}

func buildForwardKeyboard(rule model.AutoForward, buyURL string, isText bool) (*gotgbot.InlineKeyboardMarkup, error) {
	rows := make([][]gotgbot.InlineKeyboardButton, 0, 3)
	extra, err := parseForwardExtraButtons(rule.ExtraButtonsJSON)
	if err != nil {
		return nil, err
	}
	rows = append(rows, extra...)

	showBuy := buyURL != "" && ((isText && rule.BuyButtonOnText) || (!isText && rule.BuyButtonOnMedia))
	if showBuy {
		label := strings.TrimSpace(rule.BuyText)
		if label == "" {
			label = "🛒 立即购买"
		}
		rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: label, Url: buyURL}})
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}, nil
}

func parseForwardExtraButtons(raw string) ([][]gotgbot.InlineKeyboardButton, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var rows []struct {
		Text    string `json:"text"`
		URL     string `json:"url"`
		Buttons []struct {
			Text string `json:"text"`
			URL  string `json:"url"`
		} `json:"buttons"`
	}
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, fmt.Errorf("附加按钮 JSON 无效: %w", err)
	}
	out := make([][]gotgbot.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		buttons := make([]gotgbot.InlineKeyboardButton, 0, 3)
		if strings.TrimSpace(row.Text) != "" && strings.TrimSpace(row.URL) != "" {
			buttons = append(buttons, gotgbot.InlineKeyboardButton{Text: strings.TrimSpace(row.Text), Url: strings.TrimSpace(row.URL)})
		}
		for _, button := range row.Buttons {
			if strings.TrimSpace(button.Text) == "" || strings.TrimSpace(button.URL) == "" {
				continue
			}
			buttons = append(buttons, gotgbot.InlineKeyboardButton{Text: strings.TrimSpace(button.Text), Url: strings.TrimSpace(button.URL)})
		}
		if len(buttons) > 0 {
			out = append(out, buttons)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------- 过滤

func shouldSkipForward(rule model.AutoForward, b *gotgbot.Bot, msg *gotgbot.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Chat.Type == "channel" && msg.Chat.Id == 0 {
		return "invalid_chat"
	}
	text := forwardMessageText(msg)
	mediaType := forwardMediaType(msg)
	size := forwardMediaSize(msg)

	if rule.SkipForwarded && msg.ForwardOrigin != nil {
		return "forwarded"
	}
	if rule.SkipNoMedia && mediaType == "" {
		return "no_media"
	}
	if rule.SkipNoText && strings.TrimSpace(text) == "" {
		return "no_text"
	}
	if rule.MinTextLength > 0 && len([]rune(text)) < rule.MinTextLength {
		return "text_too_short"
	}
	if keywords := splitForwardKeywords(rule.SkipKeywords); len(keywords) > 0 {
		lower := strings.ToLower(text)
		for _, keyword := range keywords {
			if strings.Contains(lower, strings.ToLower(keyword)) {
				return "keyword:" + keyword
			}
		}
	}
	if size > 0 {
		if rule.MinFileSizeMB > 0 && size < int64(rule.MinFileSizeMB)*1000*1000 {
			return "file_too_small"
		}
		if rule.MaxFileSizeMB > 0 && size > int64(rule.MaxFileSizeMB)*1000*1000 {
			return "file_too_large"
		}
	}
	return ""
}

func shouldSkipForwardPreview(rule model.AutoForward, req api.AutoForwardPreviewRequest) string {
	if rule.SkipForwarded && req.IsForwarded {
		return "forwarded"
	}
	if rule.SkipNoMedia && strings.TrimSpace(req.MediaType) == "" {
		return "no_media"
	}
	if rule.SkipNoText && strings.TrimSpace(req.Text) == "" {
		return "no_text"
	}
	if rule.MinTextLength > 0 && len([]rune(req.Text)) < rule.MinTextLength {
		return "text_too_short"
	}
	if keywords := splitForwardKeywords(rule.SkipKeywords); len(keywords) > 0 {
		lower := strings.ToLower(req.Text)
		for _, keyword := range keywords {
			if strings.Contains(lower, strings.ToLower(keyword)) {
				return "keyword:" + keyword
			}
		}
	}
	if req.FileSize > 0 {
		if rule.MinFileSizeMB > 0 && req.FileSize < int64(rule.MinFileSizeMB)*1000*1000 {
			return "file_too_small"
		}
		if rule.MaxFileSizeMB > 0 && req.FileSize > int64(rule.MaxFileSizeMB)*1000*1000 {
			return "file_too_large"
		}
	}
	return ""
}

func splitForwardKeywords(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n' || r == ';' || r == '；'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// ---------------------------------------------------------------- 工具

func forwardMessageText(msg *gotgbot.Message) string {
	if msg == nil {
		return ""
	}
	if strings.TrimSpace(msg.Text) != "" {
		return msg.Text
	}
	return msg.Caption
}

func forwardMediaType(msg *gotgbot.Message) string {
	if msg == nil {
		return ""
	}
	switch {
	case len(msg.Photo) > 0:
		return "photo"
	case msg.Video != nil:
		return "video"
	case msg.Document != nil:
		return "document"
	case msg.Audio != nil:
		return "audio"
	case msg.Voice != nil:
		return "voice"
	case msg.Animation != nil:
		return "animation"
	case msg.VideoNote != nil:
		return "video_note"
	case msg.Sticker != nil:
		return "sticker"
	case msg.Story != nil:
		return "story"
	case msg.PaidMedia != nil:
		return "paid_media"
	default:
		return ""
	}
}

func forwardMediaSize(msg *gotgbot.Message) int64 {
	if msg == nil {
		return 0
	}
	switch {
	case msg.Video != nil:
		return msg.Video.FileSize
	case msg.Document != nil:
		return msg.Document.FileSize
	case msg.Audio != nil:
		return msg.Audio.FileSize
	case msg.Voice != nil:
		return msg.Voice.FileSize
	case msg.Animation != nil:
		return msg.Animation.FileSize
	case msg.VideoNote != nil:
		return msg.VideoNote.FileSize
	case len(msg.Photo) > 0:
		return msg.Photo[len(msg.Photo)-1].FileSize
	default:
		return 0
	}
}

func sanitizeForwardText(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
}

func dropSourceLines(text string) string {
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if forwardSourceLineRegex.MatchString(line) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func collapseSpacesPerLine(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(strings.ReplaceAll(line, "  ", " "))
	}
	return strings.Join(lines, "\n")
}

func forwardTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return truncateRunes(trimmed, 60)
		}
	}
	return ""
}

func slugifyForwardText(text string) string {
	first := truncateRunes(forwardTitle(text), 50)
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(first) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastDash = false
		case r >= 0x4e00 && r <= 0x9fff:
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func telegramMessageLink(chatID, messageID int64) string {
	if chatID == 0 || messageID == 0 {
		return ""
	}
	internal := -chatID
	if internal > 1000000000000 {
		return fmt.Sprintf("https://t.me/c/%d/%d", internal-1000000000000, messageID)
	}
	return ""
}

func truncateRunes(text string, limit int) string {
	if limit <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}

func isMessageNotFound(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not found") ||
		strings.Contains(message, "message_id_invalid") ||
		strings.Contains(message, "message to copy") ||
		strings.Contains(message, "message to forward")
}

func retryAfterSeconds(err error) int64 {
	if err == nil {
		return 0
	}
	var telegramErr *gotgbot.TelegramError
	if errors.As(err, &telegramErr) && telegramErr.ResponseParams != nil && telegramErr.ResponseParams.RetryAfter > 0 {
		return telegramErr.ResponseParams.RetryAfter
	}
	if match := forwardRetryAfterRegex.FindStringSubmatch(err.Error()); len(match) > 1 {
		if seconds, convErr := strconv.ParseInt(match[1], 10, 64); convErr == nil {
			return seconds
		}
	}
	return 0
}
